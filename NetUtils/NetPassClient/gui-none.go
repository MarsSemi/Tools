//go:build linux
// +build linux

package main

import (
	"fmt"
	"strings"
)

// -------------------------

// -------------------------
type GUI struct {
}

// -------------------------
func createGUI() *GUI {

	_info := "Now, you can access your local http service or websocket service.\nHere is the link : \n\n%s\n\n%s\n\n"
	_url := Global.config.Host + "/pass/" + Global.hwID + "/{local_port}"
	_wss := strings.Replace(_url, "https", "wss", 1)

	fmt.Println(fmt.Sprintf(_info, _url, _wss))

	return &GUI{}
}

// -------------------------
// 實作空的 Run，讓程式能進入 select{} 阻塞
func (_this *GUI) Run() {
	// Linux 若無 GUI，則在主線程保持阻塞
	select {}
}

// -------------------------
