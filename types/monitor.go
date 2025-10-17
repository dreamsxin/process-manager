package types

import (
	"time"
)

// ProcessStats 进程资源使用统计
type ProcessStats struct {
	PID           int       `json:"pid"`
	Name          string    `json:"name"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	MemoryBytes   uint64    `json:"memory_bytes"`
	CreateTime    time.Time `json:"create_time"`
	Timestamp     time.Time `json:"timestamp"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled         bool          `json:"enabled"`
	Interval        time.Duration `json:"interval"`
	HistorySize     int           `json:"history_size"`
	RetentionDays   int           `json:"retention_days"`
	AlertThresholds struct {
		CPU    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
		Disk   float64 `json:"disk"`
	} `json:"alert_thresholds"`
}

// ProcessMonitor 进程监控器
type ProcessMonitor struct {
	StatsHistory map[int][]ProcessStats `json:"stats_history"`
	Config       MonitorConfig          `json:"config"`
	Running      bool                   `json:"running"`
	stopChan     chan struct{}
}
