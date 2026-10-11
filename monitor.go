package main

import (
	"log"
	"sync"
	"time"
)

// TempStatus 温度状态
type TempStatus struct {
	Temp   float64 `json:"temp"`
	Level  string  `json:"level"`  // 过冷 / 正常 / 较热 / 过热 / 未知
	Color  string  `json:"color"`  // 前端配色
}

// 各档配色
const (
	colorCold   = "#5db1ff" // 过冷-冰蓝
	colorNormal = "#3ddc84" // 正常-翠绿
	colorWarm   = "#ffb020" // 较热-琥珀
	colorHot    = "#ff4d4f" // 过热-赤红
	colorUnknown = "#888888" // 未知-灰
)

// evaluate 按 GPU 用户给定标准分级
// <Cold=过冷 | Cold~Normal=正常 | Normal~Warm=较热 | >Warm=过热
func (t TempThresholds) evaluate(temp float64) TempStatus {
	if temp <= 0 {
		return TempStatus{Temp: temp, Level: "未知", Color: colorUnknown}
	}
	if temp < t.Cold {
		return TempStatus{Temp: temp, Level: "过冷", Color: colorCold}
	}
	if temp < t.Normal {
		return TempStatus{Temp: temp, Level: "正常", Color: colorNormal}
	}
	if temp < t.Warm {
		return TempStatus{Temp: temp, Level: "较热", Color: colorWarm}
	}
	return TempStatus{Temp: temp, Level: "过热", Color: colorHot}
}

// ───────────────────── 监控器 ─────────────────────

// Monitor 全局硬件监控器
type Monitor struct {
	cfg     Config
	mu      sync.RWMutex
	cpu     CPUInfo
	gpus    []GPUInfo
	mem     MemInfo
	disks   []DiskInfo
	host    HostInfo
	cpuTemp TempStatus
	gpuTemp []TempStatus
	updated time.Time
}

// NewMonitor 创建监控器
func NewMonitor(cfg Config) *Monitor {
	return &Monitor{cfg: cfg}
}

// Start 启动后台采集循环
func (m *Monitor) Start() {
	// 立即采集一次
	m.collect()
	ticker := time.NewTicker(time.Duration(m.cfg.RefreshSeconds) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.collect()
	}
}

// collect 执行一轮全量采集
func (m *Monitor) collect() {
	cpuInfo := getCPUInfo()
	gpus := getGPUInfo()
	memInfo := getMemInfo()
	diskInfo := getDiskInfo()
	hostInfo := getHostInfo()
	cpuTempRaw := getCPUThermal()

	// 评估温度
	cpuTS := m.cfg.CPU.evaluate(cpuTempRaw)
	var gpuTS []TempStatus
	for _, g := range gpus {
		gpuTS = append(gpuTS, m.cfg.GPU.evaluate(g.Temp))
	}

	m.mu.Lock()
	m.cpu = cpuInfo
	m.gpus = gpus
	m.mem = memInfo
	m.disks = diskInfo
	m.host = hostInfo
	m.cpuTemp = cpuTS
	m.gpuTemp = gpuTS
	m.updated = time.Now()
	m.mu.Unlock()

	log.Printf("[监控] CPU=%.1f%% | GPU: %d 块 | 内存=%.1f%% | 更新于 %s",
		cpuInfo.Usage, len(gpus), memInfo.UsagePct, m.updated.Format("15:04:05"))
}

// Snapshot 返回当前只读快照
func (m *Monitor) Snapshot() Overview {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Overview{
		CPU:     m.cpu,
		CPUTemp: m.cpuTemp,
		GPUs:    m.gpus,
		GPUTemp: m.gpuTemp,
		Mem:     m.mem,
		Disks:   m.disks,
		Host:    m.host,
		Updated: m.updated.Format("2006-01-02 15:04:05"),
	}
}

// Overview API 返回的总览结构
type Overview struct {
	CPU     CPUInfo      `json:"cpu"`
	CPUTemp TempStatus   `json:"cpu_temp"`
	GPUs    []GPUInfo    `json:"gpus"`
	GPUTemp []TempStatus `json:"gpu_temp"`
	Mem     MemInfo      `json:"memory"`
	Disks   []DiskInfo   `json:"disks"`
	Host    HostInfo     `json:"host"`
	Updated string       `json:"updated"`
}
