package main

//-------------------------
import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
)

// -------------------------
const defaultAppName = "NetPass Client"
const defaultHost = "https://netpass.mars-cloud.com"

// -------------------------
type GlobalData struct {
	hwID   string
	config Config
	//ui     *GUI
}

// -------------------------
var Global GlobalData

// -------------------------
// HttpRequestPayload 定義了從 Broker 接收到的 MQTT 請求資料結構
type HttpRequestPayload struct {
	Action     string              `json:"action"`      // 動作 (空或 "tunnel")
	Token      string              `json:"token"`       // 隧道識別碼
	TargetPort string              `json:"target_port"` // 目標本地 Port
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

	// 處理隧道請求 (WebSocket)
	if payload.Action == "tunnel" {
		go handleTunnel(payload)
		return
	}

	// 1. 解析與顯示請求資訊
	//pretty, _ := json.MarshalIndent(payload, "", "  ")
	//fmt.Printf("--------------------------------------------------\n")
	//fmt.Printf("Received HTTP Request via MQTT [%s]:\n%s\n", msg.Topic(), string(pretty))

	// 2. 使用傳來的 Port 與 Path
	port := payload.TargetPort
	targetPath := payload.URL

	localURL := fmt.Sprintf("http://localhost:%s%s", port, targetPath)
	//fmt.Printf("Proxying to local : %s %s\n", payload.Method, localURL)

	// 3. 建立並執行本地 HTTP 請求
	req, err := http.NewRequest(payload.Method, localURL, bytes.NewBufferString(payload.Body))
	if err != nil {
		fmt.Printf("Failed to create request : %v\n", err)
		return
	}

	// 複製與清洗 Header
	for k, vv := range payload.Header {
		_kl := strings.ToLower(k)
		if _kl == "host" {
			continue
		}
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// 強制設定 Host 為 localhost，避免被 Web Server 拒絕
	req.Host = "localhost"

	// 執行本地端 API 呼叫
	// 設定支援 Insecure TLS 的 Client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// 嘗試 HTTP 呼叫
	resp, err := httpClient.Do(req)

	// 如果 HTTP 失敗，嘗試切換為 HTTPS
	// 增加偵測 "EOF" 錯誤
	_isHTTPSOnly := false
	if err != nil {
		_errStr := err.Error()
		if strings.Contains(_errStr, "EOF") ||
			strings.Contains(_errStr, "connection refused") ||
			strings.Contains(_errStr, "http: server gave HTTP response to HTTPS client") ||
			strings.Contains(_errStr, "malformed HTTP response") {
			_isHTTPSOnly = true
		}
	}

	if _isHTTPSOnly {
		localURL = fmt.Sprintf("https://localhost:%s%s", port, targetPath)
		req, _ = http.NewRequest(payload.Method, localURL, bytes.NewBufferString(payload.Body))

		// 重新設定清洗後的 Header
		for k, vv := range payload.Header {
			_kl := strings.ToLower(k)
			if _kl == "host" {
				continue
			}
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
		req.Host = "localhost"

		resp, err = httpClient.Do(req)
	}

	// 準備回傳資料
	var responsePayload HttpResponsePayload
	responsePayload.HardwareID = Global.hwID
	responsePayload.RequestURL = localURL
	responsePayload.SessionID = payload.SessionID // 關鍵：帶回 session_id 供伺服器配對

	if err != nil {
		fmt.Printf("Local request failed : %v\n", err)
		responsePayload.Status = "Error: " + err.Error()
		responsePayload.StatusCode = 502
	} else {
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Failed to read response body: %v\n", err)
			responsePayload.Status = "Error reading response"
			responsePayload.StatusCode = 502
		} else {
			// 判斷是否為文字類資料
			contentType := resp.Header.Get("Content-Type")
			isText := strings.Contains(contentType, "text") ||
				strings.Contains(contentType, "json") ||
				strings.Contains(contentType, "javascript") ||
				strings.Contains(contentType, "xml") ||
				strings.Contains(contentType, "html")

			responsePayload.Status = resp.Status
			responsePayload.StatusCode = resp.StatusCode
			responsePayload.Header = resp.Header

			if !isText && len(respBody) > 0 {
				// 非文字類資料，轉為 Base64
				responsePayload.Body = base64.StdEncoding.EncodeToString(respBody)
				// 在 Header 中註明 Base64 格式
				responsePayload.Header["Content-Transfer-Encoding"] = []string{"base64"}
				if contentType != "" {
					responsePayload.Header["Content-Type"] = []string{contentType + "; base64"}
				}
			} else {
				responsePayload.Body = string(respBody)
			}
		}
	}

	// 4. 將執行結果發布回 MQTT (使用 http/response 前綴)
	responseTopic := fmt.Sprintf("http/response/%s", Global.hwID)
	jsonResp, err := json.Marshal(responsePayload)
	if err != nil {
		fmt.Printf("Failed to marshal response: %v\n", err)
		return
	}
	token := client.Publish(responseTopic, 1, false, jsonResp)
	token.Wait()

	//fmt.Printf("Published response with SessionID: %s\n", payload.SessionID)
}

// -------------------------
// connectHandler 連線成功時觸發
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to NetPass Tunnel")

	// 顯示公網存取網址
	_host := strings.TrimSuffix(Global.config.Host, "/")
	fmt.Printf("Public access URL: %s/pass/%s/\n", _host, Global.hwID)

	// 訂閱專屬於此硬體 ID 的請求主題 (配合 MarsCloud 規則)
	topic := fmt.Sprintf("http/request/%s", Global.hwID)
	fmt.Printf("Subscribing to topic: %s...\n", topic)

	// 使用非同步方式訂閱，並設定超時，避免卡死連線執行緒
	go func() {
		token := client.Subscribe(topic, 1, nil)
		if token.WaitTimeout(10 * time.Second) {
			if token.Error() != nil {
				fmt.Printf("Subscribe failed: %v\n", token.Error())
			} else {
				fmt.Printf("Subscribed to topic: %s\n", topic)
				//sysTray.SetStatus("Connected")
			}
		} else {
			fmt.Println("Subscribe timed out. Will retry automatically by library or next connect.")
			//sysTray.SetStatus("Create tunnel FAIL")
		}
	}()
}

// -------------------------
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v. Waiting for auto-reconnect...\n", err)
	//sysTray.SetStatus("Disconnect")
}

// -------------------------
// checkUpdate 檢查並執行自動更新
func checkUpdate() {
	_execPath, _err := os.Executable()
	if _err != nil {
		return
	}

	// 檢查是否為 go run 模式 (源碼執行)
	if strings.Contains(_execPath, "go-build") || strings.Contains(_execPath, "/tmp/") {
		fmt.Println("Source code execution (go run) detected. Skipping auto-update.")
		return
	}

	fmt.Println("Checking for updates...")
	_os := runtime.GOOS
	_arch := runtime.GOARCH

	_updateURL := fmt.Sprintf("%s/update/%s/%s", strings.TrimSuffix(Global.config.Host, "/"), _os, _arch)

	_client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	_resp, err := _client.Get(_updateURL)
	if err != nil {
		fmt.Printf("Update check failed: %v\n", err)
		return
	}
	defer _resp.Body.Close()

	if _resp.StatusCode != http.StatusOK {
		// fmt.Printf("No update available or server error (%d)\n", _resp.StatusCode)
		return
	}

	// 這裡簡單用 Content-Length 或 ETag 判斷是否需要更新
	_fInfo, err := os.Stat(_execPath)
	if err == nil {
		// fmt.Printf("Current size: %d, Remote size: %d\n", _fInfo.Size(), _resp.ContentLength)
		if _fInfo.Size() == _resp.ContentLength {
			fmt.Println("Already up to date.")
			return
		}
	}

	// 下載到臨時檔案
	_tmpPath := _execPath + ".tmp"
	_out, err := os.Create(_tmpPath)
	if err != nil {
		fmt.Printf("Failed to create temp file: %v\n", err)
		return
	}

	_, err = io.Copy(_out, _resp.Body)
	_out.Close()
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		os.Remove(_tmpPath) // 清理臨時檔案
		return
	}

	// 替換執行檔
	err = os.Chmod(_tmpPath, 0755)
	if err != nil {
		fmt.Printf("Failed to set permissions: %v\n", err)
		os.Remove(_tmpPath) // 清理臨時檔案
		return
	}

	err = os.Rename(_tmpPath, _execPath)
	if err != nil {
		fmt.Printf("Failed to replace executable: %v\n", err)
		os.Remove(_tmpPath) // 清理臨時檔案
		return
	}

	fmt.Println("Update downloaded. Restarting...")
	os.Exit(0)
}

// -------------------------
// Config 定義設定檔結構
type Config struct {
	ApiKey     string `json:"api_key"`
	Host       string `json:"host"`
	AutoUpdate bool   `json:"auto_update"`
}

// -------------------------
// loadConfig 從 config.json 載入設定
func loadConfig() {
	_file, err := os.Open("config.json")
	if err != nil {
		// 設定預設值
		if Global.config.Host == "" {
			Global.config.Host = defaultHost
		}
		return
	}
	defer _file.Close()

	decoder := json.NewDecoder(_file)
	err = decoder.Decode(&Global.config)
	if err != nil {
		fmt.Printf("Error decoding config.json: %v\n", err)
	}

	// 設定預設值
	if Global.config.Host == "" {
		Global.config.Host = defaultHost
	}
}

// -------------------------
// getAssignedID 向伺服器請求分配的連線 ID
func getAssignedID(localHWID string) string {
	_apiKey := Global.config.ApiKey
	if _apiKey == "" {
		_apiKey = os.Getenv("NETPASS_KEY") // 仍保留環境變數作為備援
	}

	_url := fmt.Sprintf("%s/api/getID", strings.TrimSuffix(Global.config.Host, "/"))

	// 準備 POST Form 資料
	_formData := strings.NewReader(fmt.Sprintf("hwid=%s&key=%s", localHWID, _apiKey))

	_client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	_resp, err := _client.Post(_url, "application/x-www-form-urlencoded", _formData)
	if err != nil {
		fmt.Printf("Failed to get assigned ID from server: %v. Using local HWID.\n", err)
		return localHWID
	}
	defer _resp.Body.Close()

	if _resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned error when assigning ID (%d). Using local HWID.\n", _resp.StatusCode)
		return localHWID
	}

	_body, err := io.ReadAll(_resp.Body)
	if err != nil {
		fmt.Printf("Failed to read assigned ID: %v\n", err)
		return localHWID
	}

	_assignedID := strings.TrimSpace(string(_body))
	if _assignedID == "" {
		fmt.Println("Server returned empty ID. Using local HWID.")
		return localHWID
	}

	return _assignedID
}

// -------------------------
// handleTunnel 建立與伺服器的 WSS 隧道並對接本地服務 (WebSocket 訊息中轉模式)
func handleTunnel(payload HttpRequestPayload) {
	// 1. 解析目標伺服器位址 (從 config.Host 提取 domain)
	_domain := Global.config.Host
	_domain = strings.TrimPrefix(_domain, "https://")
	_domain = strings.TrimPrefix(_domain, "http://")

	// 如果帶有路徑，先去掉路徑
	if _idx := strings.Index(_domain, "/"); _idx != -1 {
		_domain = _domain[:_idx]
	}

	// 分離 Host 與 Port，只取 Host
	_hostOnly := _domain
	if _idx := strings.Index(_domain, ":"); _idx != -1 {
		_hostOnly = _domain[:_idx]
	}

	// 2. 直接使用傳來的 Port 與 Path
	_port := payload.TargetPort
	_targetPath := payload.URL

	// 3. 連線到伺服器的 WSS 隧道埠 (18884)
	// 這裡必須使用 _hostOnly 避免出現 test.com:8080:18884 的錯誤
	_tunnelURL := fmt.Sprintf("wss://%s:18884/tunnel?token=%s", _hostOnly, payload.Token)
	_dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_wsTunnel, _, err := _dialer.Dial(_tunnelURL, nil)
	if err != nil {
		fmt.Printf("[Tunnel] Connection failed: %v\n", err)
		return
	}
	defer _wsTunnel.Close()

	// 4. 連線到本地 OpenClaw (WS)
	_localURL := fmt.Sprintf("ws://localhost:%s%s", _port, _targetPath)
	fmt.Printf("[Local] Connecting to %s\n", _localURL)

	_header := make(http.Header)
	for k, v := range payload.Header {
		_kl := strings.ToLower(k)
		if _kl == "upgrade" || _kl == "connection" || strings.HasPrefix(_kl, "sec-websocket-") || _kl == "host" {
			continue
		}
		_header[k] = v
	}

	// 強制覆蓋 Origin 為本地，避免被 OpenClaw 拒絕
	_header.Set("Origin", fmt.Sprintf("http://localhost:%s", _port))

	_wsLocal, _resp, err := _dialer.Dial(_localURL, _header)

	// 如果 WS 失敗，嘗試 WSS
	// 邏輯與 HTTP 失敗改 HTTPS 完全一致
	_isWSSOnly := false
	if err != nil {
		_errStr := err.Error()
		if strings.Contains(_errStr, "EOF") ||
			strings.Contains(_errStr, "connection refused") ||
			strings.Contains(_errStr, "malformed HTTP response") ||
			strings.Contains(_errStr, "unexpected EOF") {
			_isWSSOnly = true
		}
	}

	if _isWSSOnly {
		_localURL = fmt.Sprintf("wss://localhost:%s%s", _port, _targetPath)
		fmt.Printf("[Local] Fallback connecting to %s\n", _localURL)
		_header.Set("Origin", fmt.Sprintf("https://localhost:%s", _port))
		_wsLocal, _resp, err = _dialer.Dial(_localURL, _header)
	}

	if err != nil {
		if _resp != nil {
			_body, _ := io.ReadAll(_resp.Body)
			fmt.Printf("[Local] Connection failed: %v, Status: %d, Body: %s\n", err, _resp.StatusCode, string(_body))
		} else {
			fmt.Printf("[Local] Connection failed: %v\n", err)
		}
		return
	}
	defer _wsLocal.Close()
	fmt.Printf("[Local] Connected successfully to %s\n", _localURL)

	// 5. 雙向中轉 WebSocket 訊息
	_errChan := make(chan error, 2)

	// Local -> Tunnel
	go func() {
		for {
			mt, data, err := _wsLocal.ReadMessage()
			if err != nil {
				_errChan <- err
				return
			}
			err = _wsTunnel.WriteMessage(mt, data)
			if err != nil {
				_errChan <- err
				return
			}
		}
	}()

	// Tunnel -> Local
	go func() {
		for {
			mt, data, err := _wsTunnel.ReadMessage()
			if err != nil {
				_errChan <- err
				return
			}
			err = _wsLocal.WriteMessage(mt, data)
			if err != nil {
				_errChan <- err
				return
			}
		}
	}()

	<-_errChan
}

// -------------------------
// wsNetConn 將 websocket.Conn 包裝成 net.Conn 的轉接器
type wsNetConn struct {
	Conn   *websocket.Conn
	reader io.Reader
}

// -------------------------
func (c *wsNetConn) Read(b []byte) (n int, err error) {
	if c.reader == nil {
		_, c.reader, err = c.Conn.NextReader()
		if err != nil {
			return 0, err
		}
	}
	for {
		n, err = c.reader.Read(b)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			_, c.reader, err = c.Conn.NextReader()
			if err != nil {
				return 0, err
			}
			continue
		}
		return n, err
	}
}

// -------------------------
func (c *wsNetConn) Write(b []byte) (n int, err error) {
	err = c.Conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// -------------------------
func (c *wsNetConn) Close() error {
	return c.Conn.Close()
}

// -------------------------
func (c *wsNetConn) LocalAddr() net.Addr                { return c.Conn.LocalAddr() }
func (c *wsNetConn) RemoteAddr() net.Addr               { return c.Conn.RemoteAddr() }
func (c *wsNetConn) SetDeadline(t time.Time) error      { return nil }
func (c *wsNetConn) SetReadDeadline(t time.Time) error  { return c.Conn.SetReadDeadline(t) }
func (c *wsNetConn) SetWriteDeadline(t time.Time) error { return c.Conn.SetWriteDeadline(t) }

// -------------------------
// killExistingInstances 搜尋並終止其他正在運作的 NetPassClient 進程
func killExistingInstances() {
	currentPid := os.Getpid()

	// 使用 pgrep 搜尋包含 "NetPassClient" 的進程 ID
	// -f: 搜尋完整命令行
	out, err := exec.Command("pgrep", "-f", "NetPassClient").Output()
	if err != nil {
		return // 通常代表沒找到
	}

	pids := strings.Fields(string(out))
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// 排除目前進程
		if pid == currentPid {
			continue
		}

		// 取得進程並嘗試終止
		process, err := os.FindProcess(pid)
		if err == nil {
			fmt.Printf("Detected another NetPassClient (PID: %d). Killing it...\n", pid)
			process.Signal(syscall.SIGKILL)
			// 給予一點時間釋放資源
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// -------------------------
func createTunnel() {

	_localHWID := getHardwareID()
	if _localHWID == "" {
		fmt.Println("Failed to generate local hardware ID. Exiting...")
		return
	}

	// 向伺服器領取分配的 ID (可能是固定的或隨日期變動的)
	Global.hwID = getAssignedID(_localHWID)
	var clientId = fmt.Sprintf("%s", Global.hwID)

	fmt.Printf("NetPassClient starting with Assigned ID: %s\n", Global.hwID)
	//sysTray.SetHwID(hwID)
	//sysTray.SetStatus("Connecting")

	// 解析 Host 取得網域名稱以用於 MQTT (預設 18883)
	_domain := Global.config.Host
	_domain = strings.TrimPrefix(_domain, "https://")
	_domain = strings.TrimPrefix(_domain, "http://")

	// 只取 Host 部分，過濾掉 Port 與 Path
	if _idx := strings.Index(_domain, ":"); _idx != -1 {
		_domain = _domain[:_idx]
	}
	if _idx := strings.Index(_domain, "/"); _idx != -1 {
		_domain = _domain[:_idx]
	}

	_broker := fmt.Sprintf("ssl://%s:18883", _domain)
	fmt.Printf("Connecting to Tunnel: %s\n", _broker)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(_broker)
	opts.SetClientID(clientId)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	opts.SetTLSConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)

	// 設定連線中斷與重連的回呼
	opts.OnReconnecting = func(client mqtt.Client, options *mqtt.ClientOptions) {
		fmt.Println("Attempting to reconnect to NetPass Tunnel...")
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Initial connection failed: %v. Retrying in background...\n", token.Error())
	}

	//sysTray.SetStatus("Connected")
}

// -------------------------
func checkDaemon() {

	isDaemon := flag.Bool("d", false, "run in background")
	flag.Parse()

	// 2. 如果不是在背景模式，且不是 Windows (Windows 建議用編譯參數)
	if !*isDaemon && runtime.GOOS != "windows" {
		args := append(os.Args[1:], "-d")
		cmd := exec.Command(os.Args[0], args...)

		// 關鍵：將子進程與目前的 Console 脫離
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil

		err := cmd.Start()
		if err != nil {
			fmt.Printf("Cannot run as Daemon : %v\n", err)
		} else {

			fmt.Printf("NetPass run as Daemon : %d\n", cmd.Process.Pid)
			time.Sleep(1 * time.Second)
			os.Exit(0) // 結束父進程，釋放 Console
		}
	}
}

// -------------------------
func main() {

	// 啟動前先清理其他進程
	killExistingInstances()

	// 載入設定檔
	loadConfig()

	// 啟動時檢查更新
	if Global.config.AutoUpdate {
		checkUpdate()
	}

	go func() {
		checkDaemon()
		createTunnel()
	}()

	select {}
	//Global.ui = createGUI()
	//Global.ui.Run()
}

//-------------------------
