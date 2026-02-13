//go:build windows || darwin
// +build windows darwin

package main

// -------------------------
import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// -------------------------
type GUI struct {
	app    fyne.App
	window fyne.Window
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

// -------------------------
//
//go:embed icon.png
var iconData []byte

// -------------------------
func createGUI() *GUI {

	if IsEnvHasGUI() {

		_gui := &GUI{}
		_gui.app = app.NewWithID("com.marscloud.netpassclient")
		_gui.window = _gui.app.NewWindow("NetPass Client")

		if _desk, ok := _gui.app.(desktop.App); ok {

			_resIcon := fyne.NewStaticResource("./icon.png", iconData)
			_menu := fyne.NewMenu("MainMenu",
				fyne.NewMenuItem("System Info", func() {
					_gui.ShowInfo()
				}),
				fyne.NewMenuItemSeparator(), // 分隔線
				fyne.NewMenuItem("About", func() {
					_gui.ShowAbout()
				}),
			)

			_desk.SetSystemTrayMenu(_menu)
			_desk.SetSystemTrayIcon(_resIcon)
		}

		// 4. 修改關閉行為：點擊視窗 [X] 時不要結束程式，而是隱藏
		_gui.window.SetCloseIntercept(func() {
			_gui.window.Hide()
		})

		return _gui
	}

	return nil
}

// -------------------------
func (_this *GUI) Run() {

	if IsEnvHasGUI() {
		_this.app.Run()
	}
}

// -------------------------
func (_this *GUI) ShowAbout() {

	_u, _err := url.Parse("https://github.com/MarsSemi/Tools/tree/main/NetUtils/NetPassClient")
	_githubLink := widget.NewHyperlink("NetPassClient GitHub", _u)
	_since := widget.NewLabel("© since 2026 Mars Cloud")

	if _err != nil {
	}

	_githubLink.Alignment = fyne.TextAlignCenter
	_since.Alignment = fyne.TextAlignCenter
	_info := "Upon execution, this program will configure a public network tunnel,\nconnectivity between local services and the external network."

	_content := container.NewVBox(

		widget.NewLabel("About\n\n"+_info),
		widget.NewSeparator(),
		widget.NewLabel("Allcated ID : "+Global.hwID+"\nMore detail : "),
		_githubLink,
		widget.NewLabel(""),
		widget.NewSeparator(),
		_since,
	)

	_this.window.SetContent(_content)
	_this.window.Resize(fyne.NewSize(300, 200))
	_this.window.Show()
}

// -------------------------
func (_this *GUI) ShowInfo() {

	_info := "Now, you can access your local http service or websocket service.\n\nHttps link : \n%s\n\nWSS link :\n%s\n\n"
	_url := Global.config.Host + "/pass/" + Global.hwID + "/{local_port}"
	_wss := strings.Replace(_url, "https", "wss", 1)

	_content := container.NewVBox(
		widget.NewLabel("Pass-Through URL"),
		widget.NewLabel("Allcated ID : "+Global.hwID),
		widget.NewLabel(fmt.Sprintf(_info, _url, _wss)),
	)

	_this.window.SetContent(_content)
	_this.window.Resize(fyne.NewSize(300, 200))
	_this.window.Show()
}
