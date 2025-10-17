package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dreamsxin/process-manager/types"
)

// SystemMonitor 系统监控器
type SystemMonitor struct {
	history  []types.SystemStats
	config   types.MonitorConfig
	running  bool
	stopChan chan struct{}
	mu       sync.RWMutex
	dataFile string
	alerts   []string
}

// NewSystemMonitor 创建新的系统监控器
func NewSystemMonitor(dataDir string) *SystemMonitor {
	if dataDir == "" {
		dataDir = "./monitor_data"
	}

	// 确保数据目录存在
	os.MkdirAll(dataDir, 0755)

	monitor := &SystemMonitor{
		history:  make([]types.SystemStats, 0),
		stopChan: make(chan struct{}),
		dataFile: filepath.Join(dataDir, "system_stats.json"),
		alerts:   make([]string, 0),
	}

	// 默认配置
	monitor.config.Enabled = true
	monitor.config.Interval = 10 * time.Second
	monitor.config.HistorySize = 1000
	monitor.config.RetentionDays = 7
	monitor.config.AlertThresholds.CPU = 80.0
	monitor.config.AlertThresholds.Memory = 85.0
	monitor.config.AlertThresholds.Disk = 90.0

	// 加载历史数据
	monitor.loadHistory()

	return monitor
}

// Start 启动系统监控
func (sm *SystemMonitor) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("system monitor is already running")
	}

	sm.running = true
	go sm.monitoringLoop()

	return nil
}

// Stop 停止系统监控
func (sm *SystemMonitor) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return fmt.Errorf("system monitor is not running")
	}

	close(sm.stopChan)
	sm.running = false

	// 保存数据
	sm.saveHistory()

	return nil
}

// GetCurrentStats 获取当前系统统计
func (sm *SystemMonitor) GetCurrentStats() (*types.SystemStats, error) {
	return sm.collectStats()
}

// GetHistory 获取历史数据
func (sm *SystemMonitor) GetHistory(count int) []types.SystemStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if count <= 0 || count > len(sm.history) {
		count = len(sm.history)
	}

	// 返回最新的数据
	start := len(sm.history) - count
	result := make([]types.SystemStats, count)
	copy(result, sm.history[start:])

	return result
}

// GetChartData 获取图表数据
func (sm *SystemMonitor) GetChartData(count int, metric string) (*types.ChartData, error) {
	history := sm.GetHistory(count)
	if len(history) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	chartData := &types.ChartData{
		Labels:   make([]string, len(history)),
		Datasets: make([]types.Dataset, 0),
	}

	// 准备时间标签
	for i, stat := range history {
		chartData.Labels[i] = stat.Timestamp.Format("15:04:05")
	}

	// 根据指标类型准备数据
	switch metric {
	case "cpu":
		chartData.Datasets = append(chartData.Datasets, types.Dataset{
			Label:           "CPU Usage (%)",
			Data:            extractCPUData(history),
			BorderColor:     "rgb(75, 192, 192)",
			BackgroundColor: "rgba(75, 192, 192, 0.2)",
			Fill:            true,
		})
	case "memory":
		chartData.Datasets = append(chartData.Datasets, types.Dataset{
			Label:           "Memory Usage (%)",
			Data:            extractMemoryData(history),
			BorderColor:     "rgb(255, 99, 132)",
			BackgroundColor: "rgba(255, 99, 132, 0.2)",
			Fill:            true,
		})
	case "disk":
		chartData.Datasets = append(chartData.Datasets, types.Dataset{
			Label:           "Disk Usage (%)",
			Data:            extractDiskData(history),
			BorderColor:     "rgb(153, 102, 255)",
			BackgroundColor: "rgba(153, 102, 255, 0.2)",
			Fill:            true,
		})
	case "load":
		chartData.Datasets = []types.Dataset{
			{
				Label:           "Load 1min",
				Data:            extractLoad1Data(history),
				BorderColor:     "rgb(255, 159, 64)",
				BackgroundColor: "rgba(255, 159, 64, 0.2)",
				Fill:            false,
			},
			{
				Label:           "Load 5min",
				Data:            extractLoad5Data(history),
				BorderColor:     "rgb(54, 162, 235)",
				BackgroundColor: "rgba(54, 162, 235, 0.2)",
				Fill:            false,
			},
			{
				Label:           "Load 15min",
				Data:            extractLoad15Data(history),
				BorderColor:     "rgb(201, 203, 207)",
				BackgroundColor: "rgba(201, 203, 207, 0.2)",
				Fill:            false,
			},
		}
	case "all":
		chartData.Datasets = []types.Dataset{
			{
				Label:           "CPU (%)",
				Data:            extractCPUData(history),
				BorderColor:     "rgb(75, 192, 192)",
				BackgroundColor: "rgba(75, 192, 192, 0.2)",
				Fill:            false,
			},
			{
				Label:           "Memory (%)",
				Data:            extractMemoryData(history),
				BorderColor:     "rgb(255, 99, 132)",
				BackgroundColor: "rgba(255, 99, 132, 0.2)",
				Fill:            false,
			},
			{
				Label:           "Disk (%)",
				Data:            extractDiskData(history),
				BorderColor:     "rgb(153, 102, 255)",
				BackgroundColor: "rgba(153, 102, 255, 0.2)",
				Fill:            false,
			},
		}
	default:
		return nil, fmt.Errorf("unknown metric: %s", metric)
	}

	return chartData, nil
}

// GetAlerts 获取告警信息
func (sm *SystemMonitor) GetAlerts() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]string, len(sm.alerts))
	copy(result, sm.alerts)

	return result
}

// GetConfig 获取配置
func (sm *SystemMonitor) GetConfig() types.MonitorConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.config
}

// UpdateConfig 更新配置
func (sm *SystemMonitor) UpdateConfig(config types.MonitorConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if config.Interval < time.Second {
		return fmt.Errorf("monitor interval must be at least 1 second")
	}
	if config.HistorySize < 10 {
		return fmt.Errorf("history size must be at least 10")
	}

	sm.config = config

	// 如果历史数据超过新的限制，进行裁剪
	if len(sm.history) > sm.config.HistorySize {
		sm.history = sm.history[len(sm.history)-sm.config.HistorySize:]
	}

	return nil
}

// monitoringLoop 监控循环
func (sm *SystemMonitor) monitoringLoop() {
	ticker := time.NewTicker(sm.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			stats, err := sm.collectStats()
			if err != nil {
				fmt.Printf("Error collecting system stats: %v\n", err)
				continue
			}

			sm.mu.Lock()
			sm.history = append(sm.history, *stats)

			// 保持历史记录不超过配置的大小
			if len(sm.history) > sm.config.HistorySize {
				sm.history = sm.history[1:]
			}

			// 检查告警
			sm.checkAlerts(stats)

			// 定期保存数据
			if len(sm.history)%10 == 0 {
				sm.saveHistory()
			}

			sm.mu.Unlock()
		}
	}
}

// checkAlerts 检查告警条件
func (sm *SystemMonitor) checkAlerts(stats *types.SystemStats) {
	timestamp := stats.Timestamp.Format("2006-01-02 15:04:05")

	if stats.CPUPercent > sm.config.AlertThresholds.CPU {
		alert := fmt.Sprintf("[%s] CPU usage is high: %.2f%%", timestamp, stats.CPUPercent)
		sm.alerts = append(sm.alerts, alert)
	}

	if stats.MemoryPercent > sm.config.AlertThresholds.Memory {
		alert := fmt.Sprintf("[%s] Memory usage is high: %.2f%%", timestamp, stats.MemoryPercent)
		sm.alerts = append(sm.alerts, alert)
	}

	if stats.DiskPercent > sm.config.AlertThresholds.Disk {
		alert := fmt.Sprintf("[%s] Disk usage is high: %.2f%%", timestamp, stats.DiskPercent)
		sm.alerts = append(sm.alerts, alert)
	}

	// 保持告警列表大小
	if len(sm.alerts) > 100 {
		sm.alerts = sm.alerts[len(sm.alerts)-100:]
	}
}

// loadHistory 加载历史数据
func (sm *SystemMonitor) loadHistory() {
	data, err := os.ReadFile(sm.dataFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Error loading history: %v\n", err)
		}
		return
	}

	var history types.SystemStatsHistory
	if err := json.Unmarshal(data, &history); err != nil {
		fmt.Printf("Error parsing history: %v\n", err)
		return
	}

	sm.history = history.Stats

	// 应用保留策略
	sm.applyRetentionPolicy()
}

// saveHistory 保存历史数据
func (sm *SystemMonitor) saveHistory() {
	history := types.SystemStatsHistory{
		Stats: sm.history,
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling history: %v\n", err)
		return
	}

	if err := os.WriteFile(sm.dataFile, data, 0644); err != nil {
		fmt.Printf("Error saving history: %v\n", err)
	}
}

// applyRetentionPolicy 应用数据保留策略
func (sm *SystemMonitor) applyRetentionPolicy() {
	if sm.config.RetentionDays <= 0 {
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -sm.config.RetentionDays)
	var filtered []types.SystemStats

	for _, stat := range sm.history {
		if stat.Timestamp.After(cutoffTime) {
			filtered = append(filtered, stat)
		}
	}

	sm.history = filtered
}

// 数据提取辅助函数
func extractCPUData(history []types.SystemStats) []float64 {
	result := make([]float64, len(history))
	for i, stat := range history {
		result[i] = stat.CPUPercent
	}
	return result
}

func extractMemoryData(history []types.SystemStats) []float64 {
	result := make([]float64, len(history))
	for i, stat := range history {
		result[i] = stat.MemoryPercent
	}
	return result
}

func extractDiskData(history []types.SystemStats) []float64 {
	result := make([]float64, len(history))
	for i, stat := range history {
		result[i] = stat.DiskPercent
	}
	return result
}

func extractLoad1Data(history []types.SystemStats) []float64 {
	result := make([]float64, len(history))
	for i, stat := range history {
		result[i] = stat.Load1
	}
	return result
}

func extractLoad5Data(history []types.SystemStats) []float64 {
	result := make([]float64, len(history))
	for i, stat := range history {
		result[i] = stat.Load5
	}
	return result
}

func extractLoad15Data(history []types.SystemStats) []float64 {
	result := make([]float64, len(history))
	for i, stat := range history {
		result[i] = stat.Load15
	}
	return result
}
