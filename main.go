//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"unsafe"
)

func main() {
	// GUI 必须固定在主 OS 线程
	runtime.LockOSThread()
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	fmt.Println("H武器 启动中...")

	// 加载配置
	gConfig = loadConfig()
	gTimerMs = uint32(gConfig.RefreshSeconds) * 1000
	if gTimerMs < 500 {
		gTimerMs = 2000
	}
	log.Printf("[配置] CPU阈值=%.0f/%.0f/%.0f°C  GPU阈值=%.0f/%.0f/%.0f°C  刷新=%dms",
		gConfig.CPU.Cold, gConfig.CPU.Normal, gConfig.CPU.Warm,
		gConfig.GPU.Cold, gConfig.GPU.Normal, gConfig.GPU.Warm, gTimerMs)

	// 启动后台采集
	gMonitor = NewMonitor(gConfig)
	go gMonitor.Start()

	// DPI 感知（避免高 DPI 下模糊），失败可忽略
	pSetProcessDPIAware.Call()

	// 获取模块实例
	hInst, _, _ := pGetModuleHandle.Call(0)
	gInstance = hInst

	// 注册窗口类
	if err := registerWindowClass(gInstance); err != nil {
		log.Fatalf("[错误] 注册窗口类失败: %v", err)
	}

	// 创建窗口
	hwnd := createMainWindow(gInstance)
	if hwnd == 0 {
		log.Fatalf("[错误] 创建窗口失败")
	}
	gHwnd = hwnd

	// 显示窗口
	pShowWindow.Call(hwnd, uintptr(SW_SHOW))
	pUpdateWindow.Call(hwnd)

	log.Println("[就绪] H武器 主窗口已显示")

	// 进入消息循环
	code := messageLoop()
	os.Exit(code)
}

// 保留 unsafe 引用以避免未使用导入（部分字段使用 unsafe.Pointer）
var _ = unsafe.Sizeof(0)
