package types

import (
	"time"
)

// SystemStats 系统资源使用统计
type SystemStats struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total"`
	DiskPercent   float64   `json:"disk_percent,omitempty"`
	DiskUsed      uint64    `json:"disk_used,omitempty"`
	DiskTotal     uint64    `json:"disk_total,omitempty"`
	Load1         float64   `json:"load_1,omitempty"`
	Load5         float64   `json:"load_5,omitempty"`
	Load15        float64   `json:"load_15,omitempty"`
}

// SystemStatsHistory 系统统计历史记录
type SystemStatsHistory struct {
	Stats []SystemStats `json:"stats"`
}

// ChartData 图表数据
type ChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset 数据集
type Dataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BorderColor     string    `json:"borderColor,omitempty"`
	BackgroundColor string    `json:"backgroundColor,omitempty"`
	Fill            bool      `json:"fill,omitempty"`
}
