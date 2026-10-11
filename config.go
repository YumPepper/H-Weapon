package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// TempThresholds 温度分档阈值
// 低于 Cold = 过冷 | Cold~Normal = 正常 | Normal~Warm = 较热 | 高于 Warm = 过热
type TempThresholds struct {
	Cold   float64 `json:"cold"`
	Normal float64 `json:"normal"`
	Warm   float64 `json:"warm"`
}

// Config 全局配置
type Config struct {
	CPU            TempThresholds `json:"cpu"`
	GPU            TempThresholds `json:"gpu"`
	RefreshSeconds int            `json:"refresh_seconds"`
	Port           int            `json:"port"`
}

// defaultConfig 返回默认配置（GPU 阈值遵循用户要求）
func defaultConfig() Config {
	return Config{
		CPU:            TempThresholds{Cold: 20, Normal: 75, Warm: 85},
		GPU:            TempThresholds{Cold: 30, Normal: 80, Warm: 90},
		RefreshSeconds: 2,
		Port:           9527,
	}
}

// loadConfig 读取 exe 同目录下的 config.json，缺失时使用默认值并写回
func loadConfig() Config {
	cfg := defaultConfig()
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[WARN] 无法获取 exe 路径: %v, 使用默认配置", err)
		return cfg
	}
	cfgPath := filepath.Join(filepath.Dir(exePath), "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// 尝试当前工作目录
		data, err = os.ReadFile("config.json")
	}
	if err == nil {
		// 合并：解析出的字段覆盖默认值
		var loaded Config
		if json.Unmarshal(data, &loaded) == nil {
			mergeConfig(&cfg, loaded)
		}
	} else {
		// 配置文件不存在，写回默认配置
		writeDefaultConfig(cfgPath, cfg)
	}
	return cfg
}

func mergeConfig(dst *Config, src Config) {
	if src.CPU.Cold > 0 {
		dst.CPU = src.CPU
	}
	if src.GPU.Cold > 0 {
		dst.GPU = src.GPU
	}
	if src.RefreshSeconds > 0 {
		dst.RefreshSeconds = src.RefreshSeconds
	}
	if src.Port > 0 {
		dst.Port = src.Port
	}
}

func writeDefaultConfig(path string, cfg Config) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}
