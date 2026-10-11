//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// CREATE_NO_WINDOW：子进程不创建新的控制台窗口，
// 防止 GUI 程序调用 nvidia-smi 等外部命令时弹出 CMD 黑窗。
const createNoWindow = 0x08000000

// hideCmdWindow 隐藏子进程的控制台窗口（仅 Windows 有效）
func hideCmdWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
