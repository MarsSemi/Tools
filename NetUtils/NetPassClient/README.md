# NetPassClient (HTTP Inverse Proxy Client)

NetPassClient 是 NetPass 架構的內網執行端。
使用這個程式後，就可以透過系統配置的 Https Domain，透過 HTTPs 協議遠端存取你寫的 Web Service。
原理是連接到遠端的 NetPassService，並監聽專屬於本機硬體 ID 的請求訊息，將其轉換為本地 HTTP 呼叫後回傳結果。
讓你無須申請 Domain Name 以及網路 SSL 憑證，就可安全地讓你的服務遠端運作。

## 核心功能
*   **硬體識別 (HWID)**：自動產生 12 位元 MAC 雜湊 ID，無需手動配置識別碼。
*   **反向代理 (Inverse Proxy)**：接收遠端請求並轉發至本地指定 Port (例如 localhost:8080)。
*   **自動重連**：具備強健的自動重連機制，即使網路不穩也能在恢復後自動訂閱主題。
*   **雙向傳輸**：執行本地 API 後，將結果 (Header, Body, Status) 自動打包回傳至伺服器。
*   **隔離環境**：每個 Client 擁有獨立的主題路徑，互不干擾。

## 系統需求
*   Go 1.25 或更高版本
*   可連接至 `netpass.mars-cloud.com` 的網路環境

## 快速佈署教程

### 1. 安裝依賴
在專案目錄下執行：
```bash
go mod tidy
```

### 2. 啟動 Client
```bash
# 直接啟動
go run .
# 您會看到類似以下的輸出：
# NetPassClient starting with ID: f60358672553
# Connected to NetPass Tunnel
# Subscribed to topic: http/request/f60358672553
```

### 3. 如何接收請求
一旦 Client 啟動，外部使用者只需存取以下 URL 格式：
`https://netpass.mars-cloud.com/pass/{CLIENT_ID}/{LOCAL_PORT}/{API or PATH}`

範例：
若您的 ID 為 `f60358672553`，且您本地 9000 port 有一個 HTTP 服務：
`https://netpass.mars-cloud.com/pass/f60358672553/9000/api/hello`

### 4. 原始程式背景運行 (Linux)
```bash
nohup go run . > client.log 2>&1 &
```

### 5. 編譯可執行檔
```bash
./run.sh
# 則在 bin 目錄下產生可執行檔
# 請依平台選擇執行檔案
```

## 目錄結構
*   `main.go`：代理轉發邏輯與 MQTT 用戶端實作。
*   `go.mod`：專案依賴管理。
*   `README.md`：本說明文件。
