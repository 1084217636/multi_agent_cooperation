//go:build tray

package ui

import (
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

// TrayApp 提供可选的系统托盘入口。
type TrayApp struct {
	dashboardURL string
	onQuit       func()
}

// NewTrayApp 创建新的托盘应用。
func NewTrayApp(dashboardURL string, onQuit func()) *TrayApp {
	return &TrayApp{
		dashboardURL: dashboardURL,
		onQuit:       onQuit,
	}
}

// Run 运行托盘应用。
func (ta *TrayApp) Run() error {
	systray.Run(ta.onReady, ta.onExit)
	return nil
}

func (ta *TrayApp) onReady() {
	systray.SetIcon([]byte{})
	systray.SetTitle("Desk Companion")
	systray.SetTooltip("Go 研发智能体工作台")

	openItem := systray.AddMenuItem("打开工作台", "打开研发智能体工作台")
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("退出", "退出工作台")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				_ = openDashboard(ta.dashboardURL)
			case <-quitItem.ClickedCh:
				if ta.onQuit != nil {
					ta.onQuit()
				}
				systray.Quit()
				return
			}
		}
	}()
}

func (ta *TrayApp) onExit() {}

func openDashboard(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
