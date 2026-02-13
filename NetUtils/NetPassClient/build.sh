#!/bin/bash
go mod tidy
# 設定輸出的二進位檔名稱
APP_NAME="NetPassClient"
OUTPUT_DIR="./bin"

# 建立輸出目錄
mkdir -p $OUTPUT_DIR

echo "開始編譯 NetPass Client..."

# 定義要編譯的平台
# 格式: "GOOS/GOARCH/EXTENSION"
PLATFORMS=(
    "windows/amd64/.exe"
    "linux/amd64/"
    "linux/arm64/"
    "darwin/arm64/"
)

# 獲取 Go 的完整路徑
GO_BIN=$(which go)
if [ -z "$GO_BIN" ]; then
    GO_BIN="/usr/local/go/bin/go"
fi

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS="/" read -r OS ARCH EXT <<< "$PLATFORM"
    
    # 根據需求將 amd64 改名為 x64
    FILENAME_ARCH=$ARCH
    if [ "$ARCH" == "amd64" ]; then
        FILENAME_ARCH="x64"
    fi

    # 根據平台設定編譯環境
    if [ "$OS" == "linux" ]; then
        export CGO_ENABLED=0
        unset CC
    elif [ "$OS" == "windows" ]; then
        export CGO_ENABLED=1
        export CC=x86_64-w64-mingw32-gcc
    elif [ "$OS" == "darwin" ]; then
        export CGO_ENABLED=1
        unset CC
    fi
    
    OUTPUT_NAME="${APP_NAME}_${OS}_${FILENAME_ARCH}${EXT}"
    echo "正在編譯: $OS/$ARCH -> $OUTPUT_DIR/$OUTPUT_NAME"
    
    # 執行編譯 (加上 -ldflags 以隱藏 Windows 的命令提示字元視窗)
    if [ "$OS" == "windows" ]; then
        env GOOS=$OS GOARCH=$ARCH go build -ldflags="-H windowsgui" -o "$OUTPUT_DIR/$OUTPUT_NAME" .
    else
        env GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_DIR/$OUTPUT_NAME" .
    fi
    
    if [ $? -eq 0 ]; then
        echo "✅ $OS/$ARCH 編譯成功"
    else
        echo "❌ $OS/$ARCH 編譯失敗"
    fi
done

echo "------------------------------------"
echo "所有平台編譯完成，檔案位於: $OUTPUT_DIR"
ls -F $OUTPUT_DIR
