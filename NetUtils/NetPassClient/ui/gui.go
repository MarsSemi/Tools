package main

// -------------------------
import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// -------------------------
type GUI struct {
	app    fyne.App
	window fyne.Window
}

// -------------------------
func createGUI() *GUI {

	_gui := &GUI{}
	_gui.app = app.NewWithID("com.marscloud.netpassclient")
	_gui.window = _gui.app.NewWindow("NetPass Client")

	if _desk, ok := _gui.app.(desktop.App); ok {

		_menu := fyne.NewMenu("MainMenu",
			fyne.NewMenuItem("顯示主視窗", func() {
				_gui.ShowInfo()
			}),
			fyne.NewMenuItemSeparator(), // 分隔線
			fyne.NewMenuItem("關於", func() {
				_gui.ShowAbout()
			}),
		)

		_desk.SetSystemTrayMenu(_menu)
		_desk.SetSystemTrayIcon(theme.FyneLogo())
	}

	// 4. 修改關閉行為：點擊視窗 [X] 時不要結束程式，而是隱藏
	_gui.window.SetCloseIntercept(func() {
		_gui.window.Hide()
	})

	return _gui
}

// -------------------------
func (_this *GUI) Run() {
	_this.app.Run()
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

	_content := container.NewVBox(
		widget.NewLabel("關於\n本程式執行後將自動配置一個公用網路，將外網的請求透傳至本地服務。"),
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

	_content := container.NewVBox(
		widget.NewLabel("這是一個跨平台托盤程式"),
	)

	_this.window.SetContent(_content)
	_this.window.Resize(fyne.NewSize(300, 200))
	_this.window.Show()
}
