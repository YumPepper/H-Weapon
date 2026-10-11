//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// ───────────────────── Win32 DLL ─────────────────────

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	pRegisterClassEx = user32.NewProc("RegisterClassExW")
	pCreateWindowEx  = user32.NewProc("CreateWindowExW")
	pDefWindowProc   = user32.NewProc("DefWindowProcW")
	pShowWindow      = user32.NewProc("ShowWindow")
	pUpdateWindow    = user32.NewProc("UpdateWindow")
	pGetMessage      = user32.NewProc("GetMessageW")
	pDispatchMessage = user32.NewProc("DispatchMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pSetTimer        = user32.NewProc("SetTimer")
	pPostQuitMessage = user32.NewProc("PostQuitMessage")
	pInvalidateRect  = user32.NewProc("InvalidateRect")
	pBeginPaint      = user32.NewProc("BeginPaint")
	pEndPaint        = user32.NewProc("EndPaint")
	pGetClientRect   = user32.NewProc("GetClientRect")
	pLoadCursor       = user32.NewProc("LoadCursorW")
	pFillRect         = user32.NewProc("FillRect")
	pDrawTextW        = user32.NewProc("DrawTextW")
	pSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")

	pGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")

	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pCreateFontW       = gdi32.NewProc("CreateFontW")
	pTextOutW          = gdi32.NewProc("TextOutW")
)

// ───────────────────── Win32 常量 ─────────────────────

const (
	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001

	WS_OVERLAPPED      = 0x00000000
	WS_CAPTION         = 0x00C00000
	WS_SYSMENU         = 0x00080000
	WS_THICKFRAME      = 0x00040000
	WS_MINIMIZEBOX     = 0x00020000
	WS_MAXIMIZEBOX     = 0x00010000
	WS_OVERLAPPEDWINDOW = WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX

	SW_SHOW = 5
	IDC_ARROW = 32512

	WM_CREATE      = 0x0001
	WM_DESTROY     = 0x0002
	WM_PAINT       = 0x000F
	WM_ERASEBKGND  = 0x0014
	WM_TIMER       = 0x0113

	TRANSPARENT       = 1
	FW_NORMAL         = 400
	FW_BOLD           = 700
	DEFAULT_CHARSET   = 1
	CLEARTYPE_QUALITY = 5
	DT_LEFT           = 0x00000020

	CW_USEDEFAULT = 0x80000000
)

// ───────────────────── Win32 结构体 (内存布局与 C 一致) ─────────────────────

type rect struct {
	left, top, right, bottom int32
}

type point struct {
	x, y int32
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra   int32
	cbWndExtra   int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

// ───────────────────── 工具函数 ─────────────────────

// rgb 生成 COLORREF (0x00BBGGRR)
func rgb(r, g, b int) uintptr {
	return uintptr(r | (g << 8) | (b << 16))
}

func utf16Ptr(s string) (*uint16, error) {
	return syscall.UTF16PtrFromString(s)
}

// registerWindowClass 注册窗口类
func registerWindowClass(hInstance uintptr) error {
	className, _ := utf16Ptr("HWeaponWndClass")
	cursor, _, _ := pLoadCursor.Call(0, uintptr(IDC_ARROW))
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		style:         CS_HREDRAW | CS_VREDRAW,
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInstance,
		hCursor:       cursor,
		hbrBackground: 0, // 自行绘制背景
		lpszClassName: className,
	}
	r, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return err
	}
	return nil
}

// createMainWindow 创建主窗口
func createMainWindow(hInstance uintptr) uintptr {
	title, _ := utf16Ptr("H武器 — 硬件检测 & 温度监测  v1.0.0")
	className, _ := utf16Ptr("HWeaponWndClass")
	hwnd, _, _ := pCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(WS_OVERLAPPEDWINDOW),
		uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT),
		920, 880,
		0, 0, hInstance, 0,
	)
	return hwnd
}

// messageLoop 消息循环
func messageLoop() int {
	for {
		var m msg
		r, _, _ := pGetMessage.Call(
			uintptr(unsafe.Pointer(&m)),
			0, 0, 0,
		)
		if int32(r) == 0 { // WM_QUIT
			return int(m.wParam)
		}
		if int32(r) == -1 { // 错误
			return -1
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}
