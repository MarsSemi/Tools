# NetPassClient (智慧適應版)

執行於內網環境，自動將公網請求轉發至本地服務。

## 核心特性
*   **自動協議偵測**：自動判斷本地服務為 HTTP 還是 HTTPS，遇到 502 或連線拒絕時會自動切換協議。
*   **二進位傳輸**：自動偵測 `Content-Type`，若非文字資料則以 Base64 編碼回傳。
*   **雙向傳輸**：支援普通 HTTP 請求與 WebSocket 升級請求。

## 配置說明 (`config.json`)
```json
{
  "api_key": "your_key",
  "host": "https://test.mars-cloud.com:8080"
}
```

## 運行建議
開發環境直接使用 `go run .` 執行，若需常駐請搭配系統服務管理。
目前 Assigned ID 會在啟動時印出，或從 `client.log` 中查看。
