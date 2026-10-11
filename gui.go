//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// ───────────────────── 全局资源 ─────────────────────

var (
	gInstance  uintptr
	gHwnd      uintptr
	gMonitor   *Monitor
	gConfig    Config
	gTimerMs   uint32 = 2000

	gFontTitle   uintptr // HFONT
	gFontHeader  uintptr
	gFontBody    uintptr
	gFontMono    uintptr
	gBrushBg     uintptr
	gBrushSection uintptr
	gBrushLine   uintptr

	// 配色
	colLabel = rgb(195, 205, 215)
	colValue = rgb(235, 240, 248)
	colDim   = rgb(140, 150, 165)
	colTitle = rgb(255, 205, 70)
	colAccent = rgb(255, 205, 70)
)

const (
	marginX = 28
	lineH   = 22
)

// ───────────────────── 窗口过程 ─────────────────────

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		createResources()
		pSetTimer.Call(hwnd, uintptr(timerID), uintptr(gTimerMs), 0)
		return 0

	case WM_ERASEBKGND:
		// 不让系统擦除背景，避免闪烁；我们在 WM_PAINT 中自行填充
		return 1

	case WM_TIMER:
		if wparam == uintptr(timerID) {
			pInvalidateRect.Call(hwnd, 0, 0) // erase=FALSE，避免闪烁
		}
		return 0

	case WM_PAINT:
		var ps paintStruct
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		drawDashboard(hdc)
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_DESTROY:
		if gFontTitle != 0 {
			pDeleteObject.Call(gFontTitle)
		}
		if gFontHeader != 0 {
			pDeleteObject.Call(gFontHeader)
		}
		if gFontBody != 0 {
			pDeleteObject.Call(gFontBody)
		}
		if gFontMono != 0 {
			pDeleteObject.Call(gFontMono)
		}
		if gBrushBg != 0 {
			pDeleteObject.Call(gBrushBg)
		}
		if gBrushSection != 0 {
			pDeleteObject.Call(gBrushSection)
		}
		if gBrushLine != 0 {
			pDeleteObject.Call(gBrushLine)
		}
		pPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := pDefWindowProc.Call(hwnd, msg, wparam, lparam)
	return ret
}

// ───────────────────── 资源创建 ─────────────────────

func createResources() {
	faceYa, _ := utf16Ptr("微软雅黑")
	faceCons, _ := utf16Ptr("Consolas")

	gFontTitle = createFont(-32, FW_BOLD, faceYa)
	gFontHeader = createFont(-22, FW_BOLD, faceYa)
	gFontBody = createFont(-19, FW_NORMAL, faceYa)
	gFontMono = createFont(-18, FW_NORMAL, faceCons)

	gBrushBg = createBrush(rgb(18, 22, 30))
	gBrushSection = createBrush(rgb(30, 36, 50))
	gBrushLine = createBrush(rgb(55, 65, 85))
}

const timerID = 1001

func createFont(height int32, weight int, face *uint16) uintptr {
	h, _, _ := pCreateFontW.Call(
		uintptr(height), 0, 0, 0,
		uintptr(weight), 0, 0, 0,
		uintptr(DEFAULT_CHARSET), 0, 0,
		uintptr(CLEARTYPE_QUALITY), 0,
		uintptr(unsafe.Pointer(face)),
	)
	return h
}

func createBrush(color uintptr) uintptr {
	h, _, _ := pCreateSolidBrush.Call(color)
	return h
}

// ───────────────────── 绘制工具 ─────────────────────

// drawText 在指定位置绘制文本
func drawText(hdc uintptr, x, y int32, text string, font uintptr, color uintptr) {
	pSelectObject.Call(hdc, font)
	pSetTextColor.Call(hdc, color)
	pSetBkMode.Call(hdc, uintptr(TRANSPARENT))
	w := syscall.StringToUTF16(text)
	if len(w) == 0 {
		return
	}
	pTextOutW.Call(hdc, uintptr(x), uintptr(y),
		uintptr(unsafe.Pointer(&w[0])), uintptr(len(w)-1))
}

// drawSectionHeader 绘制分段标题栏，返回内容起始 y
func drawSectionHeader(hdc uintptr, y int32, title string, w int32) int32 {
	var rc rect
	rc.left = marginX
	rc.top = y
	rc.right = w - marginX
	rc.bottom = y + 32
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), gBrushSection)
	drawText(hdc, marginX+12, y+7, title, gFontHeader, colAccent)
	return y + 42
}

// drawHLine 绘制水平细线
func drawHLine(hdc uintptr, y, x1, x2 int32) {
	var rc rect
	rc.left = x1
	rc.top = y
	rc.right = x2
	rc.bottom = y + 1
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), gBrushLine)
}

// levelColor 根据温度等级返回对应颜色
func levelColor(level string) uintptr {
	switch level {
	case "过冷":
		return rgb(93, 177, 255)
	case "正常":
		return rgb(61, 220, 132)
	case "较热":
		return rgb(255, 176, 32)
	case "过热":
		return rgb(255, 77, 79)
	default:
		return rgb(150, 150, 150)
	}
}

// ───────────────────── 仪表盘主绘制 ─────────────────────

func drawDashboard(hdc uintptr) {
	var rc rect
	pGetClientRect.Call(gHwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.right - rc.left

	// 填充背景
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), gBrushBg)

	snap := gMonitor.Snapshot()
	var y int32 = 16

	// 标题
	drawText(hdc, marginX, y, "⚡ H 武器", gFontTitle, colTitle)
	drawText(hdc, 200, y+10, "硬件检测 & 温度监测工具", gFontHeader, colValue)
	y += 44
	drawText(hdc, marginX, y, "v1.0.0  ·  纯 Go 原生实现  ·  数据仅供参考", gFontBody, colDim)
	y += 26
	drawHLine(hdc, y, marginX, w-marginX)
	y += 12

	// === CPU ===
	y = drawSectionHeader(hdc, y, "CPU 处理器", w)
	drawText(hdc, marginX+14, y, "名称：", gFontBody, colLabel)
	drawText(hdc, marginX+14+drawTextWidth(hdc, "名称：", gFontBody), y, snap.CPU.Name, gFontBody, colValue)
	y += lineH
	drawText(hdc, marginX+14, y,
		fmt.Sprintf("核心：%d 物理 / %d 逻辑", snap.CPU.Cores, snap.CPU.LogicalCores), gFontBody, colLabel)
	y += lineH
	drawText(hdc, marginX+14, y,
		fmt.Sprintf("使用率：%.1f %%", snap.CPU.Usage), gFontBody, colLabel)
	y += lineH
	tc := levelColor(snap.CPUTemp.Level)
	drawText(hdc, marginX+14, y,
		fmt.Sprintf("温度：%.1f °C", snap.CPUTemp.Temp), gFontBody, tc)
	drawText(hdc, marginX+200, y, "  ［"+snap.CPUTemp.Level+"］", gFontBody, tc)
	y += lineH + 8

	// === GPU ===
	y = drawSectionHeader(hdc, y, "GPU 显卡", w)
	if len(snap.GPUs) == 0 {
		drawText(hdc, marginX+14, y, "未检测到独立 GPU（需 nvidia-smi 或受支持驱动）", gFontBody, colDim)
		y += lineH
	} else {
		for i, g := range snap.GPUs {
			ts := snap.GPUTemp[i]
			gc := levelColor(ts.Level)
			drawText(hdc, marginX+14, y, fmt.Sprintf("GPU %d：%s", i+1, g.Name), gFontBody, colValue)
			y += lineH
			drawText(hdc, marginX+30, y,
				fmt.Sprintf("驱动：%s    显存：%d MB", g.Driver, g.MemoryMB), gFontBody, colLabel)
			y += lineH
			drawText(hdc, marginX+30, y,
				fmt.Sprintf("使用率：%.1f %%    风扇：%.0f %%    功耗：%.1f W",
					g.Usage, g.FanSpeed, g.PowerDraw), gFontBody, colLabel)
			y += lineH
			drawText(hdc, marginX+30, y,
				fmt.Sprintf("温度：%.1f °C", ts.Temp), gFontBody, gc)
			drawText(hdc, marginX+260, y, "  ［"+ts.Level+"］", gFontBody, gc)
			y += lineH + 6
		}
	}
	y += 6

	// === 内存 ===
	y = drawSectionHeader(hdc, y, "内存", w)
	drawText(hdc, marginX+14, y,
		fmt.Sprintf("总量：%.1f GB    已用：%.1f GB    可用：%.1f GB",
			snap.Mem.TotalGB, snap.Mem.UsedGB, snap.Mem.AvailableGB), gFontBody, colLabel)
	y += lineH
	// 简易进度条
	drawBar(hdc, marginX+14, y+4, 520, 14, snap.Mem.UsagePct, levelColorForUsage(snap.Mem.UsagePct))
	drawText(hdc, marginX+14+530, y, fmt.Sprintf("%.1f %%", snap.Mem.UsagePct), gFontBody, colValue)
	y += lineH + 8

	// === 磁盘 ===
	y = drawSectionHeader(hdc, y, fmt.Sprintf("磁盘  (共 %d 块)", len(snap.Disks)), w)
	if len(snap.Disks) == 0 {
		drawText(hdc, marginX+14, y, "未检测到磁盘", gFontBody, colDim)
		y += lineH
	} else {
		for _, d := range snap.Disks {
			drawText(hdc, marginX+14, y,
				fmt.Sprintf("%s   (%s)   %s", d.Mountpoint, d.Fstype, d.Device), gFontBody, colValue)
			y += lineH
			drawText(hdc, marginX+30, y,
				fmt.Sprintf("总量：%.1f GB    已用：%.1f GB    使用率：%.1f %%",
					d.TotalGB, d.UsedGB, d.UsagePct), gFontBody, colLabel)
			y += lineH
			drawBar(hdc, marginX+30, y+4, 504, 12, d.UsagePct, levelColorForUsage(d.UsagePct))
			y += lineH + 6
		}
	}
	y += 6

	// === 主机 ===
	y = drawSectionHeader(hdc, y, "主机信息", w)
	drawText(hdc, marginX+14, y, "操作系统："+snap.Host.OS, gFontBody, colLabel)
	y += lineH
	drawText(hdc, marginX+14, y, "平台："+snap.Host.Platform, gFontBody, colLabel)
	y += lineH
	drawText(hdc, marginX+14, y, "内核："+snap.Host.Kernel, gFontBody, colLabel)
	y += lineH
	drawText(hdc, marginX+14, y, "主机名："+snap.Host.Hostname, gFontBody, colLabel)
	y += lineH
	drawText(hdc, marginX+14, y,
		fmt.Sprintf("运行时长：%.1f 小时", snap.Host.UptimeHours), gFontBody, colLabel)
	y += lineH + 8

	// === 温度标准 ===
	y = drawSectionHeader(hdc, y, "温度标准", w)
	drawText(hdc, marginX+14, y,
		"GPU：< 30°C 过冷 ｜ 30~80°C 正常 ｜ 80~90°C 较热 ｜ > 90°C 过热", gFontBody, colLabel)
	y += lineH
	cC, cN, cW := gConfig.CPU.Cold, gConfig.CPU.Normal, gConfig.CPU.Warm
	drawText(hdc, marginX+14, y,
		fmt.Sprintf("CPU：< %.0f°C 过冷 ｜ %.0f~%.0f°C 正常 ｜ %.0f~%.0f°C 较热 ｜ > %.0f°C 过热",
			cC, cC, cN, cN, cW, cW), gFontBody, colLabel)
	y += lineH + 6
	// 颜色图例
	drawLegend(hdc, y, marginX+14)
	y += lineH + 8

	// 页脚
	drawHLine(hdc, y, marginX, w-marginX)
	y += 8
	drawText(hdc, marginX, y, "更新时间："+snap.Updated, gFontBody, colDim)
}

// levelColorForUsage 根据使用率返回颜色（用于进度条）
func levelColorForUsage(pct float64) uintptr {
	switch {
	case pct >= 90:
		return rgb(255, 77, 79)
	case pct >= 75:
		return rgb(255, 176, 32)
	default:
		return rgb(61, 220, 132)
	}
}

// drawBar 绘制进度条
func drawBar(hdc uintptr, x, y, w, h int32, pct float64, color uintptr) {
	// 背景
	var rc rect
	rc.left = x
	rc.top = y
	rc.right = x + w
	rc.bottom = y + h
	brBg, _, _ := pCreateSolidBrush.Call(rgb(45, 52, 65))
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brBg)
	pDeleteObject.Call(brBg)
	// 填充
	if pct > 0 {
		fw := int32(float64(w) * pct / 100)
		if fw < 1 {
			fw = 1
		}
		var rc2 rect
		rc2.left = x
		rc2.top = y
		rc2.right = x + fw
		rc2.bottom = y + h
		pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc2)), color)
	}
}

// drawLegend 绘制温度颜色图例
func drawLegend(hdc uintptr, y, x int32) {
	items := []struct {
		name  string
		color uintptr
	}{
		{"过冷", rgb(93, 177, 255)},
		{"正常", rgb(61, 220, 132)},
		{"较热", rgb(255, 176, 32)},
		{"过热", rgb(255, 77, 79)},
	}
	curX := x
	for _, it := range items {
		// 色块
		var rc rect
		rc.left = curX
		rc.top = y + 3
		rc.right = curX + 18
		rc.bottom = y + 17
		br, _, _ := pCreateSolidBrush.Call(it.color)
		pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), br)
		pDeleteObject.Call(br)
		drawText(hdc, curX+24, y+1, it.name, gFontBody, colLabel)
		curX += 24 + drawTextWidth(hdc, it.name, gFontBody) + 28
	}
}

// drawTextWidth 估算文本宽度（字符数 * 平均字宽）
// 由于精确测量需 GetTextExtentPoint32W，这里用近似值，用于排版
func drawTextWidth(hdc uintptr, text string, font uintptr) int32 {
	// 简单估算：每字符约 10 像素（中文约 19，混合取 12）
	runes := []rune(text)
	est := int32(0)
	for _, r := range runes {
		if r > 127 {
			est += 19
		} else {
			est += 10
		}
	}
	return est
}
