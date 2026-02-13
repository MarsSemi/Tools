// -------------------------
package main

//-------------------------
import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/getlantern/systray"
)

// -------------------------
type GUI struct {
	menuInfo  *systray.MenuItem
	menuQuit  *systray.MenuItem
	menuAbout *systray.MenuItem

	hasGUI bool
}

// -------------------------
func createGUI() *GUI {

	_gui := &GUI{}
	_gui.hasGUI = IsEnvHasGUI()

	return _gui
}

// -------------------------
// IsEnvHasGUI 偵測目前系統是否具有 GUI 操作環境
func IsEnvHasGUI() bool {
	switch runtime.GOOS {
	case "windows":
		// 在 Windows，我們嘗試呼叫 GetProcessWindowStation
		// 雖然大多數 Windows 都有 GUI，但在 Windows Server Core 或 SSH 模式下可能不同
		// 這裡使用最簡單的判斷方式：通常 Windows 桌面環境總是具備 GUI
		return true

	case "darwin": // macOS
		// 在 macOS，如果是在終端機透過 SSH 連入，通常無法開啟 GUI App
		// 檢查是否能與 WindowServer 通訊
		cmd := exec.Command("lsappinfo", "list")
		err := cmd.Run()
		return err == nil

	case "linux":
		// Linux 主要檢查環境變數
		// DISPLAY 是 X11 的標準
		// WAYLAND_DISPLAY 是 Wayland 的標準
		if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
			return true
		}
		return false

	default:
		return false
	}
}

// ------------------------
func loadImageFromURL(url string) []byte {
	// 1. Perform an HTTP GET request to the URL
	res, err := http.Get(url)
	if err != nil {
		return nil
	}
	// Ensure the response body is closed after the function returns
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil
	}

	// 2. Decode the image data from the response body
	// image.Decode automatically detects the image format
	img, format, err := image.Decode(res.Body)
	if err != nil {
		return nil
	}

	var buf bytes.Buffer
	switch format {
	case "jpg":
	case "jpeg":
		jpeg.Encode(&buf, img, nil)
	case "png":
		png.Encode(&buf, img)
	}

	return buf.Bytes()
}

// -------------------------
func (_this *GUI) ShowDialog(_title, _content string) {

	switch runtime.GOOS {
	case "windows":
		// 使用 PowerShell 彈出訊息框
		cmd := fmt.Sprintf("Add-Type -AssemblyName PresentationFramework; [System.Windows.MessageBox]::Show('%s', '%s')", _content, _title)
		exec.Command("powershell", "-Command", cmd).Run()

	case "darwin": // macOS
		// 使用 AppleScript 顯示對話框
		cmd := fmt.Sprintf("display dialog \"%s\" with title \"%s\" buttons {\"OK\"} default button \"OK\"", _content, _title)
		exec.Command("osascript", "-e", cmd).Run()

	case "linux":
		// Linux 通常依賴 zenity 或 notify-send
		exec.Command("zenity", "--info", "--title", _title, "--text", _content, "--width=300").Run()

	default:
		fmt.Println(_content) // 無 GUI 環境則輸出至終端機
	}
}

// -------------------------
func (_this *GUI) ShowInfoDialog() {

	title := "NetPassClient : Pass-through Info"
	url := Global.config.Host + "/pass/{port}"
	wss := strings.Replace(url, "https", "wss", 1)

	// 組合要顯示的資訊
	info := fmt.Sprintf(
		"Allocated ID : %s\n\n"+"-\nPass-through URL :\n\n%s\n\n"+"-\nPass-through Websockett : \n\n%s\n",
		Global.hwID, url, wss)

	// 根據 OS 執行不同的原生彈窗指令
	_this.ShowDialog(title, info)
}

// -------------------------
func (_this *GUI) ShowAboutDialog() {
	title := "About NetPassClient"
	// 組合要顯示的資訊
	info := fmt.Sprintf(
		"Version : 0.2.12\n"+
			"Allocated ID : %s\n"+"-\n\nMore detail : \n\nhttps://github.com/MarsSemi/Tools/tree/main/NetUtils/NetPassClient\n\n"+"-\n© since 2026 Mars Cloud",
		Global.hwID)

	// 根據 OS 執行不同的原生彈窗指令
	_this.ShowDialog(title, info)
}

// -------------------------
func (_this *GUI) onGUIReady() {

	systray.SetTitle("NetPassClient")
	systray.SetTooltip("NetPass Tunneling Service")

	// 設定選單
	_this.menuInfo = systray.AddMenuItem("Information", "")
	systray.AddSeparator()

	_this.menuAbout = systray.AddMenuItem("About", "About this app")
	_this.menuQuit = systray.AddMenuItem("Quit App", "Quit the app")

	// 監聽 UI 事件
	go func() {
		for {
			select {
			case <-_this.menuQuit.ClickedCh:
				systray.Quit()
				os.Exit(0)

			case <-_this.menuInfo.ClickedCh:
				_this.ShowInfoDialog()

			case <-_this.menuAbout.ClickedCh:
				_this.ShowAboutDialog()
			}
		}
	}()

	_icon := loadImageFromURL("https://netpass.mars-cloud.com/images/netpass.png")

	if _icon != nil {
		systray.SetIcon(_icon)
		systray.SetTitle("")
	}
}

// -------------------------
func (_this *GUI) onGUIExit() {
}

// -------------------------
func (_this *GUI) Run() {

	_this.hasGUI = IsEnvHasGUI()

	if _this.hasGUI {
		systray.Run(_this.onGUIReady, _this.onGUIExit)

	} else {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
	}
}

//-------------------------
