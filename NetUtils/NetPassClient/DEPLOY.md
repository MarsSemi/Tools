# NetPassClient 部署指南

本文檔說明如何部署 NetPassClient 用戶端。

## 部署方式

### 方式一：直接執行 (推薦)

1. **選擇執行檔**
   根據你的作業系統和架構，選擇對應的執行檔：
   - Linux x64: `NetPassClient_linux_x64`
   - Linux arm64: `NetPassClient_linux_arm64`
   - macOS arm64: `NetPassClient_darwin_arm64`
   - Windows x64: `NetPassClient_windows_x64.exe`

2. **配置**
   編輯 `config.json`：
   ```json
   {
     "api_key": "your_api_key",
     "host": "https://your-server.com:443"
   }
   ```

3. **執行**
   ```bash
   # Linux/macOS
   chmod +x NetPassClient_linux_x64
   ./NetPassClient_linux_x64

   # Windows
   .\NetPassClient_windows_x64.exe
   ```

### 方式二：使用腳本

```bash
./run.sh
```

腳本會自動偵測平台並選擇正確的執行檔。

### 方式三：自行編譯

```bash
./build.sh
```

## 配置說明

### config.json 欄位

| 欄位 | 說明 | 範例 |
|------|------|------|
| `api_key` | 伺服器核發的 API Key | `abc123def456...` |
| `host` | 伺服器 URL | `https://netpass.mars-cloud.com:443` |

## 常見部署場景

### 1. 單一設備部署

適用於需要從內網暴露單一服務的情況：

1. 設定 config.json
2. 執行 Client
3. 記錄 Assigned ID
4. 透過 `https://server/pass/{id}/{port}/` 訪問

### 2. 多設備部署

適用於物聯網設備批量部署：

1. 預先配置相同的 config.json
2. 每次啟動會獲得不同的 ID（可設定固定 ID）
3. 透過管理介面查看所有設備狀態

### 3. Docker 部署

建立 Dockerfile：
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o NetPassClient .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/NetPassClient .
COPY config.json .
CMD ["./NetPassClient"]
```

建構並執行：
```bash
docker build -t netpass-client .
docker run -d --name netpass-client -v $(pwd)/config.json:/app/config.json netpass-client
```

### 4. Systemd 服務 (Linux)

建立服務檔 `/etc/systemd/system/netpass-client.service`：
```ini
[Unit]
Description=NetPass Client
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/NetPassClient
ExecStart=/home/ubuntu/NetPassClient/NetPassClient_linux_x64
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

啟動服務：
```bash
sudo systemctl daemon-reload
sudo systemctl enable netpass-client
sudo systemctl start netpass-client
```

## 驗證部署

### 檢查運行狀態

```bash
# 查看日誌
tail -f client.log

# 檢查程序
ps aux | grep NetPassClient
```

### 測試連線

1. 確認 Client 輸出顯示 `Connected to server`
2. 確認顯示 Assigned ID
3. 嘗試訪問穿透網址

## 自動化部署腳本

建立部署腳本 `deploy.sh`：
```bash
#!/bin/bash

# 停止舊服務
pkill NetPassClient || true

# 複製執行檔
cp NetPassClient_linux_x64 NetPassClient
chmod +x NetPassClient

# 啟動新服務
nohup ./NetPassClient > client.log 2>&1 &

echo "NetPassClient deployed. Assigned ID:"
grep "Assigned ID" client.log
```

## 故障排除

### 問題：連線被拒絕

**可能原因**：
- API Key 錯誤或過期
- 伺服器未運行
- 網路問題

**解決方案**：
1. 檢查 config.json 中的 api_key
2. 確認伺服器運行正常
3. 檢查網路連線

### 問題： Assigned ID 每次不同

**說明**：
這是正常行為，每次啟動會分配新 ID。

**如需固定 ID**：
请联系服务器管理员配置固定设备 ID。

### 問題：穿透訪問失敗

**可能原因**：
- 本地服務未运行
- 連接埠錯誤
- 防火牆阻擋

**解決方案**：
1. 確認本地服務正常運行
2. 檢查連接埠號是否正確
3. 檢查本地防火牆

## 後續管理

### 查看 Assigned ID

```bash
grep "Assigned ID" client.log
```

### 查看日誌

```bash
tail -f client.log
```

### 重啟服務

```bash
pkill NetPassClient
./NetPassClient_linux_x64
```

### 自動重啟

使用 systemd 或 supervisor 實現自動重啟。

---

由 塔奇克馬 (Tachikoma) 維護 🕷️
