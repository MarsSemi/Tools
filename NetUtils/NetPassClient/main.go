package main

//-------------------------
import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// -------------------------
// hwID 儲存本機硬體唯一識別碼 (12位元)
var hwID string

// -------------------------
// HttpRequestPayload 定義了從 Broker 接收到的 MQTT 請求資料結構
type HttpRequestPayload struct {
	Method     string              `json:"method"`      // HTTP 方法
	URL        string              `json:"url"`         // 包含路由資訊的路徑
	Header     map[string][]string `json:"header"`      // 轉發的 Header
	Body       string              `json:"body"`        // 請求主體
	HardwareID string              `json:"hardware_id"` // 來源 Broker ID
	SessionID  string              `json:"session_id"`  // 交易追蹤 ID
}

// -------------------------
// HttpResponsePayload 定義了要回傳給 Broker 的 MQTT 回應資料結構
type HttpResponsePayload struct {
	StatusCode int                 `json:"status_code"` // HTTP 狀態碼
	Status     string              `json:"status"`      // 狀態描述
	Header     map[string][]string `json:"header"`      // 本地端 Response Header
	Body       string              `json:"body"`        // 本地端 Response Body
	HardwareID string              `json:"hardware_id"` // 本機 Client ID
	RequestURL string              `json:"request_url"` // 被呼叫的本地 URL
	SessionID  string              `json:"session_id"`  // 對應請求的交易 ID
}

// -------------------------
// getHardwareID 透過 MAC Address 產生唯一的 12 位元硬體識別碼
func getHardwareID() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	var hardwareData string
	for _, i := range interfaces {
		if i.HardwareAddr.String() != "" {
			hardwareData += i.HardwareAddr.String()
		}
	}

	if hardwareData == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(hardwareData))
	return hex.EncodeToString(hash[:])[:12]
}

// -------------------------
// messagePubHandler 是處理 MQTT 訊息的核心函式
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var payload HttpRequestPayload
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		fmt.Printf("[RAW] Received: %s\n", msg.Payload())
		return
	}

	// 1. 解析與顯示請求資訊
	//pretty, _ := json.MarshalIndent(payload, "", "  ")
	//fmt.Printf("--------------------------------------------------\n")
	//fmt.Printf("Received HTTP Request via MQTT [%s]:\n%s\n", msg.Topic(), string(pretty))

	// 2. URL 路由解析 (解析格式: /hwid/port/path)
	path := strings.TrimPrefix(payload.URL, "/")
	parts := strings.SplitN(path, "/", 3)

	if len(parts) < 2 {
		fmt.Printf("Invalid URL format: %s (expected /hwid/port/path)\n", payload.URL)
		return
	}

	// 取得目標 Port 與 Path
	port := parts[1]
	targetPath := ""
	if len(parts) > 2 {
		targetPath = "/" + parts[2]
	}

	localURL := fmt.Sprintf("http://localhost:%s%s", port, targetPath)
	//fmt.Printf("Proxying to local : %s %s\n", payload.Method, localURL)

	// 3. 建立並執行本地 HTTP 請求
	req, err := http.NewRequest(payload.Method, localURL, bytes.NewBufferString(payload.Body))
	if err != nil {
		fmt.Printf("Failed to create request : %v\n", err)
		return
	}

	// 複製所有 Header 到本地請求
	for k, vv := range payload.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// 執行本地端 API 呼叫
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(req)

	// 準備回傳資料
	var responsePayload HttpResponsePayload
	responsePayload.HardwareID = hwID
	responsePayload.RequestURL = localURL
	responsePayload.SessionID = payload.SessionID // 關鍵：帶回 session_id 供伺服器配對

	if err != nil {
		fmt.Printf("Local request failed : %v\n", err)
		responsePayload.Status = "Error: " + err.Error()
		responsePayload.StatusCode = 502
	} else {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		//fmt.Printf("Local response : %s\n", resp.Status)

		responsePayload.Status = resp.Status
		responsePayload.StatusCode = resp.StatusCode
		responsePayload.Header = resp.Header
		responsePayload.Body = string(respBody)
	}

	// 4. 將執行結果發布回 MQTT (使用 http/response 前綴)
	responseTopic := fmt.Sprintf("http/response/%s", hwID)
	jsonResp, _ := json.Marshal(responsePayload)
	token := client.Publish(responseTopic, 1, false, jsonResp)
	token.Wait()

	//fmt.Printf("Published response with SessionID: %s\n", payload.SessionID)
}

// -------------------------
// connectHandler 連線成功時觸發
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to NetPass Tunnel")
	// 訂閱專屬於此硬體 ID 的請求主題 (配合 MarsCloud 規則)
	topic := fmt.Sprintf("http/request/%s", hwID)
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		fmt.Printf("Subscribe failed: %v\n", token.Error())
	} else {
		fmt.Printf("Subscribed to topic: %s\n", topic)
	}
}

// -------------------------
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v. Auto-reconnecting...\n", err)
}

// -------------------------
func main() {
	hwID = getHardwareID()
	var clientId = fmt.Sprintf("%s", hwID)

	if clientId == "" {
		fmt.Println("Failed to generate hardware ID. Exiting...")
		return
	}

	fmt.Printf("NetPassClient starting with ID: %s\n", clientId)

	opts := mqtt.NewClientOptions()
	opts.AddBroker("ssl://netpass.mars-cloud.com:18883")
	opts.SetClientID(clientId)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	opts.SetTLSConfig(&tls.Config{
		InsecureSkipVerify: false,
	})

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Initial connection failed: %v. Retrying in background...\n", token.Error())
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	client.Disconnect(250)
	fmt.Println("Client disconnected.")
}

//-------------------------
