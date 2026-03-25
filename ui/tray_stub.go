package ui

import "fmt"

// TrayApp 是系统托盘的默认占位实现。
// 真实托盘功能需要使用 `-tags tray` 编译。
type TrayApp struct {
	dashboardURL string
	onQuit       func()
}

// NewTrayApp 创建一个占位托盘应用。
func NewTrayApp(dashboardURL string, onQuit func()) *TrayApp {
	return &TrayApp{
		dashboardURL: dashboardURL,
		onQuit:       onQuit,
	}
}

// Run 在默认构建中提示需要 tray tag。
func (ta *TrayApp) Run() error {
	_ = ta.dashboardURL
	_ = ta.onQuit
	return fmt.Errorf("native tray requires -tags tray")
}
