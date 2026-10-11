//go:build !windows

package main

import "os/exec"

// 非 Windows 平台：空实现（保持 hardware.go 跨平台编译）
func hideCmdWindow(cmd *exec.Cmd) {}
