# NetPassClient 用戶端

運行於內網環境的穿透客戶端，自動將公網請求轉發至本地服務。

## 功能特性

### 核心功能
- **自動協議偵測**：自動判斷本地服務為 HTTP 或 HTTPS，遇到 502 時自動切換
- **二進制傳輸**：自動偵測 Content-Type，非文字資料以 Base64 編碼回傳
- **雙向傳輸**：支援普通 HTTP 請求與 WebSocket 升級請求
- **MQTT + WSS**：控制指令走 MQTT，數據傳輸走 WSS 隧道

### 支援平台
| 平台 | 架構 | 執行檔 |
|------|------|--------|
| Linux | x64 | NetPassClient_linux_x64 |
| Linux | arm64 | NetPassClient_linux_arm64 |
| macOS | arm64 | NetPassClient_darwin_arm64 |
| Windows | x64 | NetPassClient_windows_x64.exe |

## 目錄結構

```
Client/
├── NetPassClient              # 預設執行檔 (當前平台)
├── NetPassClient_linux_x64   # Linux x64
├── NetPassClient_linux_arm64  # Linux arm64
├── NetPassClient_darwin_arm64 # macOS arm64
├── NetPassClient_windows_x64.exe # Windows x64
├── config.json               # 配置文件
├── main.go                   # 入口點
├── gui.go                    # GUI 版本
├── gui-none.go               # 無 GUI 版本
├── build.sh                  # 編譯腳本
├── run.sh                    # 執行腳本
└── bin/                      # 預編譯執行檔目錄
```

## 配置說明

### config.json
```json
{
  "api_key": "your_issued_api_key",
  "host": "https://your-server.com:443",
  "name": "office-gateway-01"
}
```

### 配置項說明

| 欄位 | 說明 | 必填 |
|------|------|------|
| `api_key` | 伺服器核發的 API Key | 是 |
| `host` | 伺服器 URL (http:// 或 https://) | 是 |
| `name` | 可選設備別名，3-64 字元，需全站唯一，可用來替代 ID 存取 | 否 |

## 快速開始

### 1. 配置
編輯 `config.json`，填入伺服器核發的 API Key 和伺服器 URL：
```json
{
  "api_key": "abc123...",
  "host": "https://netpass.mars-cloud.com:443",
  "name": "office-gateway-01"
}
```

若設定 `name`，Client 啟動時會先向 Server 註冊；`name` 若重複、格式錯誤，或 Server 無法註冊成功，Client 會直接停止，不會退回成本機 HWID。

### 2. 執行

**Linux/macOS**
```bash
# 使用腳本（自動選擇正確的執行檔）
./run.sh

# 或直接執行
./bin/NetPassClient_linux_x64
```

**Windows**
```bash
.\bin\NetPassClient_windows_x64.exe
```

### 3. 驗證
執行後會顯示目前使用的 `id/name`，請記錄以供後續訪問：
```
NetPassClient starting with : abc123def456/office-gateway-01
```

## 使用方式

### 基本穿透訪問

假設：
- 伺服器網域：`netpass.mars-cloud.com`
- 設備 ID：`mydevice`
- 本地服務連接埠：`8080`

若 Client 註冊了 `name`，則下方 `mydevice` 可以直接使用該名稱。

WebSSH 也可直接使用相同的 `name` 或 `id`。

訪問本地 HTTP 服務：
```
https://netpass.mars-cloud.com/pass/mydevice/80/
https://netpass.mars-cloud.com/pass/mydevice/8080/
```

訪問本地 HTTPS 服務：
```
https://netpass.mars-cloud.com/pass/mydevice/443/
```

訪問特定路徑：
```
https://netpass.mars-cloud.com/pass/mydevice/8080/api/users
```

### WebSocket 支援

自動支援 WebSocket 連線升級，無需額外配置。

## 編譯 (可選)

如需自行編譯：

### 環境要求
- **Go**：1.18+
- **系統依賴**：
  - Linux: `libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libgl1-mesa-dev`
  - macOS: Xcode Command Line Tools
  - Windows: MinGW-w64

### 編譯命令
```bash
cd Client

# 編譯當前平台
./build.sh

# 或手動編譯
go build -o NetPassClient .

# 交叉編譯
GOOS=linux GOARCH=amd64 go build -o NetPassClient_linux_x64 .
GOOS=linux GOARCH=arm64 go build -o NetPassClient_linux_arm64 .
GOOS=darwin GOARCH=arm64 go build -o NetPassClient_darwin_arm64 .
GOOS=windows GOARCH=amd64 go build -o NetPassClient_windows_x64.exe .
```

## 常見問題

### CGO 相關錯誤

**Linux 錯誤**
```
fatal error: X11/extensions/Xrandr.h: No such file or directory
```

**解決方案**
```bash
sudo apt-get update
sudo apt-get install libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libgl1-mesa-dev
sudo apt-get install xorg-dev libxxf86vm-dev libasound2-dev libx11-dev
```

**Windows (MinGW) 錯誤**
```
cgo: C compiler "x86_64-w64-mingw32-gcc" not found
```

**解決方案**
```bash
sudo apt-get install gcc-mingw-w64
```

### 連線問題

**無法連接到伺服器**
- 確認 `config.json` 中的 `host` 正確
- 檢查網路是否可以訪問伺服器
- 確認伺服器的防火牆已開放必要連接埠

**認證失敗**
- 確認 `api_key` 正確且未過期
- 聯繫管理員重新核發 API Key
- 若有設定 `name`，確認該名稱未重複，且只包含 `a-z`、`0-9`、`.`、`_`、`-`

### 執行問題

**Permission Denied**
```bash
chmod +x NetPassClient_linux_x64
```

**找不到執行檔**
確保使用正確的平台執行檔，或使用 `./build.sh` 重新編譯。

## 與 OpenClaw 整合

NetPassClient 可以作為 OpenClaw 的 Skill 使用：

1. 將 Client 目錄製作成 Skill
2. 在 OpenClaw 中配置 api_key 和 host
3. 使用 Skill 指令啟動/停止穿透

詳情請參考 OpenClaw Skill 文件。

## 日誌說明

執行後會在同目錄生成 `client.log`，包含：
- 連線狀態
- 錯誤訊息
- 請求記錄

查看日誌：
```bash
tail -f client.log
```

---

由 塔奇克馬 (Tachikoma) 維護 🕷️
