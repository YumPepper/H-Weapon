package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// ───────────────────── 数据结构 ─────────────────────

// CPUInfo CPU 信息
type CPUInfo struct {
	Name        string  `json:"name"`
	Cores       int     `json:"cores"`
	LogicalCores int    `json:"logical_cores"`
	Frequency   float64 `json:"frequency_mhz"`
	Usage       float64 `json:"usage_percent"`
	Temp        float64 `json:"temp_celsius"`
}

// GPUInfo GPU 信息
type GPUInfo struct {
	Name      string  `json:"name"`
	Driver    string  `json:"driver_version"`
	MemoryMB  uint64  `json:"memory_mb"`
	Usage     float64 `json:"usage_percent"`
	Temp      float64 `json:"temp_celsius"`
	FanSpeed  float64 `json:"fan_speed_percent"`
	PowerDraw float64 `json:"power_watts"`
}

// MemInfo 内存信息
type MemInfo struct {
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	UsagePct    float64 `json:"usage_percent"`
}

// DiskInfo 单块磁盘信息
type DiskInfo struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Fstype     string  `json:"fstype"`
	TotalGB    float64 `json:"total_gb"`
	UsedGB     float64 `json:"used_gb"`
	FreeGB     float64 `json:"free_gb"`
	UsagePct   float64 `json:"usage_percent"`
}

// HostInfo 主机信息
type HostInfo struct {
	OS           string `json:"os"`
	Platform     string `json:"platform"`
	Kernel       string `json:"kernel"`
	Hostname     string `json:"hostname"`
	UptimeHours  float64 `json:"uptime_hours"`
}

// ───────────────────── 采集函数 ─────────────────────

func gb(b uint64) float64 { return float64(b) / 1024 / 1024 / 1024 }
func round1(f float64) float64 {
	return float64(int(f*10)) / 10
}

// getCPUInfo 采集 CPU 信息
func getCPUInfo() CPUInfo {
	info := CPUInfo{}

	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		info.Name = strings.TrimSpace(infos[0].ModelName)
		info.Cores = int(infos[0].Cores)
	}
	info.LogicalCores = runtime.NumCPU()
	info.Frequency = 0

	if usage, err := cpu.Percent(time.Second, false); err == nil && len(usage) > 0 {
		info.Usage = round1(usage[0])
	}
	return info
}

// getGPUInfo 采集 GPU 信息（优先 nvidia-smi）
func getGPUInfo() []GPUInfo {
	gpus := queryNvidiaSmi()
	if gpus != nil {
		return gpus
	}
	// 回退：返回空列表，前端显示"未检测到独立 GPU"
	return []GPUInfo{}
}

// queryNvidiaSmi 通过 nvidia-smi 获取 NVIDIA GPU 信息
func queryNvidiaSmi() []GPUInfo {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=name,driver_version,memory.total,utilization.gpu,temperature.gpu,fan.speed,power.draw",
		"--format=csv,noheader,nounits",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	hideCmdWindow(cmd) // 隐藏 CMD 窗口，防止 GUI 程序调用外部命令时弹黑窗
	if err := cmd.Run(); err != nil {
		return nil
	}
	var gpus []GPUInfo
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		g := GPUInfo{
			Name:     strings.TrimSpace(parts[0]),
			Driver:   strings.TrimSpace(parts[1]),
			MemoryMB: parseUintSafe(parts[2]),
		}
		g.Usage = parseFloatSafe(parts[3])
		g.Temp = parseFloatSafe(parts[4])
		g.FanSpeed = parseFloatSafe(parts[5])
		g.PowerDraw = parseFloatSafe(parts[6])
		gpus = append(gpus, g)
	}
	return gpus
}

func parseUintSafe(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v
}
func parseFloatSafe(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return round1(v)
}

// getMemInfo 采集内存信息
func getMemInfo() MemInfo {
	info := MemInfo{}
	if v, err := mem.VirtualMemory(); err == nil {
		info.TotalGB = round1(gb(v.Total))
		info.UsedGB = round1(gb(v.Used))
		info.AvailableGB = round1(gb(v.Available))
		info.UsagePct = round1(v.UsedPercent)
	}
	return info
}

// getDiskInfo 采集磁盘分区信息
func getDiskInfo() []DiskInfo {
	var disks []DiskInfo
	parts, err := disk.Partitions(false)
	if err != nil {
		return disks
	}
	for _, p := range parts {
		u, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		disks = append(disks, DiskInfo{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			TotalGB:    round1(gb(u.Total)),
			UsedGB:     round1(gb(u.Used)),
			FreeGB:     round1(gb(u.Free)),
			UsagePct:   round1(u.UsedPercent),
		})
	}
	return disks
}

// getHostInfo 采集主机信息
func getHostInfo() HostInfo {
	info := HostInfo{}
	h, err := host.Info()
	if err == nil {
		info.OS = h.OS
		info.Platform = fmt.Sprintf("%s %s", h.Platform, h.PlatformVersion)
		info.Kernel = h.KernelVersion
		info.Hostname = h.Hostname
		info.UptimeHours = round1(float64(h.Uptime) / 3600)
	}
	return info
}

// getCPUThermal 通过 WMI/gopsutil 传感器尝试读取 CPU 温度
func getCPUThermal() float64 {
	// gopsutil 的 SensorsTemperatures 在 Windows 下通过 WMI 读取
	// 多数主板不直接暴露 CPU 核温，返回 0 表示不可用
	return 0
}
