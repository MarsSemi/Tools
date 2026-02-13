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
如執行或編譯遇到下列錯誤：

```
# runtime/cgo
gcc: error: unrecognized command-line option '-m64'

或是

cgo: C compiler "x86_64-w64-mingw32-gcc" not found: exec: "x86_64-w64-mingw32-gcc": executable file not found in $PATH

這是因為缺少 MinGW 編譯器，請安裝：

sudo apt-get update
sudo apt-get install gcc-mingw-w64
```

```
fatal error: X11/extensions/Xrandr.h: No such file or directory

這是典型的 golang CGO (C bindings) 問題，請更新相關的lib後，
再次執行或編譯

sudo apt-get update
sudo apt-get install libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libgl1-mesa-dev
sudo apt-get install xorg-dev libxxf86vm-dev libasound2-dev libx11-dev libxcursor-dev libxinerama-dev
```