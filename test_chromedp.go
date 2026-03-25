//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	fmt.Println("Testing chromedp with Edge browser...")
	
	// 检查Edge浏览器是否存在
	edgePath := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
	if _, err := os.Stat(edgePath); os.IsNotExist(err) {
		edgePath = "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe"
		if _, err := os.Stat(edgePath); os.IsNotExist(err) {
			fmt.Printf("Edge browser not found at: %s or %s\n", 
				"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
				"C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe")
			return
		}
	}
	fmt.Printf("Using Edge browser at: %s\n", edgePath)
	
	// 创建Edge选项（使用headless模式）
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.ExecPath(edgePath),
	)
	
	fmt.Println("Creating browser context...")
	
	// 创建执行器上下文
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	
	fmt.Println("Creating Chrome context...")
	
	// 创建Chrome上下文
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	
	fmt.Println("Browser context created successfully!")
	
	// 导航到简单页面
	fmt.Println("Navigating to https://www.google.com...")
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.google.com"),
	)
	if err != nil {
		fmt.Printf("Failed to navigate: %v\n", err)
		return
	}
	
	fmt.Println("Navigation successful!")
	
	// 等待页面加载
	time.Sleep(5 * time.Second)
	
	// 获取页面标题
	var title string
	err = chromedp.Run(ctx,
		chromedp.Title(&title),
	)
	if err != nil {
		fmt.Printf("Failed to get title: %v\n", err)
		return
	}
	
	fmt.Printf("Page title: %s\n", title)
	
	fmt.Println("Test completed successfully!")
}
