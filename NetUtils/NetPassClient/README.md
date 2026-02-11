# NetPassClient

執行於內網環境，自動將公網請求轉發至本地服務。

## 核心特性
*   **自動協議偵測**：自動判斷本地服務為 HTTP 還是 HTTPS，遇到 502 或連線拒絕時會自動切換協議。
*   **二進位傳輸**：自動偵測 `Content-Type`，若非文字資料則以 Base64 編碼回傳。
*   **雙向傳輸**：支援普通 HTTP 請求與 WebSocket 升級請求。

## 配置說明 (`config.json`)
```json
{
  "api_key": "your_key",
  "host": "https://netpass.mars-cloud.com"
}
```

## 運行建議
開發環境直接使用 `go run .` 執行，若需常駐請搭配系統服務管理。
目前 Assigned ID 會在啟動時印出，或從 `client.log` 中查看。

## 運行方法
選擇你平台對應的執行檔，運行後會獲得一個 ID

使用 https://netpass.mars-cloud.com/pass/[id]/[target_port]
則會橋接你在防火牆後端的 Http Service

## 範例
如果你在 local 執行了一個 Http Service，
有個 Restful API 是 http://127.0.0.1:8080/api/hello
正確執行本程式後會配置一組 ID，該 Restful api 會連通以下網址：

# https://netpass.mars-cloud.com/pass/[id]/8080/hello

其餘透傳功能依此類推。
