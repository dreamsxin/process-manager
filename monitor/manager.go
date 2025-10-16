package monitor

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dreamsxin/process-manager/types"
)

// ProcessMonitorManager 进程监控管理器
type ProcessMonitorManager struct {
	monitoredProcesses map[int]string // pid -> name
	statsHistory       map[int][]types.ProcessStats
	config             types.MonitorConfig
	running            bool
	stopChan           chan struct{}
	mu                 sync.RWMutex
}

// NewProcessMonitorManager 创建新的进程监控管理器
func NewProcessMonitorManager() *ProcessMonitorManager {
	return &ProcessMonitorManager{
		monitoredProcesses: make(map[int]string),
		statsHistory:       make(map[int][]types.ProcessStats),
		config: types.MonitorConfig{
			Enabled:     true,
			Interval:    2 * time.Second,
			HistorySize: 60, // 保留最近60个样本
		},
		stopChan: make(chan struct{}),
	}
}

// Start 启动监控
func (m *ProcessMonitorManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("monitor is already running")
	}

	m.running = true
	go m.monitoringLoop()
	return nil
}

// Stop 停止监控
func (m *ProcessMonitorManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("monitor is not running")
	}

	close(m.stopChan)
	m.running = false
	return nil
}

// AddProcess 添加进程到监控列表
func (m *ProcessMonitorManager) AddProcess(pid int, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.monitoredProcesses[pid]; exists {
		return fmt.Errorf("process %d is already being monitored", pid)
	}

	m.monitoredProcesses[pid] = name
	m.statsHistory[pid] = make([]types.ProcessStats, 0, m.config.HistorySize)
	return nil
}

// RemoveProcess 从监控列表移除进程
func (m *ProcessMonitorManager) RemoveProcess(pid int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.monitoredProcesses[pid]; !exists {
		return fmt.Errorf("process %d is not being monitored", pid)
	}

	delete(m.monitoredProcesses, pid)
	delete(m.statsHistory, pid)
	return nil
}

// GetProcessStats 获取进程统计信息
func (m *ProcessMonitorManager) GetProcessStats(pid int) (*types.ProcessStats, error) {
	stats, err := getProcessStats(pid)
	if err != nil {
		return nil, err
	}

	// 如果进程在监控列表中，更新名称
	m.mu.RLock()
	if name, exists := m.monitoredProcesses[pid]; exists {
		stats.Name = name
	}
	m.mu.RUnlock()

	return stats, nil
}

// GetProcessStatsByName 按进程名获取统计信息
func (m *ProcessMonitorManager) GetProcessStatsByName(name string) ([]types.ProcessStats, error) {
	pids, names, err := getPIDsByName(name)
	if err != nil {
		return nil, err
	}

	var statsList []types.ProcessStats
	for i, pid := range pids {
		stats, err := getProcessStats(pid)
		if err != nil {
			continue // 忽略错误的进程
		}
		stats.Name = names[i] // 使用从系统中获取的实际进程名
		statsList = append(statsList, *stats)
	}

	return statsList, nil
}

// GetAllStats 获取所有被监控进程的统计信息
func (m *ProcessMonitorManager) GetAllStats() ([]types.ProcessStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var statsList []types.ProcessStats
	for pid, name := range m.monitoredProcesses {
		stats, err := getProcessStats(pid)
		if err != nil {
			continue // 进程可能已经退出
		}
		stats.Name = name
		statsList = append(statsList, *stats)
	}

	// 按PID排序
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].PID < statsList[j].PID
	})

	return statsList, nil
}

// GetProcessHistory 获取进程历史统计
func (m *ProcessMonitorManager) GetProcessHistory(pid int, count int) ([]types.ProcessStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.statsHistory[pid]
	if !exists {
		return nil, fmt.Errorf("no history found for process %d", pid)
	}

	if count > len(history) {
		count = len(history)
	}

	// 返回最新的数据
	start := len(history) - count
	return history[start:], nil
}

// GetConfig 获取监控配置
func (m *ProcessMonitorManager) GetConfig() types.MonitorConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// UpdateConfig 更新监控配置
func (m *ProcessMonitorManager) UpdateConfig(config types.MonitorConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Interval < time.Second {
		return fmt.Errorf("monitor interval must be at least 1 second")
	}
	if config.HistorySize < 1 {
		return fmt.Errorf("history size must be at least 1")
	}

	m.config = config
	return nil
}

// GetMonitoredProcesses 获取被监控的进程列表
func (m *ProcessMonitorManager) GetMonitoredProcesses() map[int]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[int]string)
	for pid, name := range m.monitoredProcesses {
		result[pid] = name
	}
	return result
}

// monitoringLoop 监控循环
func (m *ProcessMonitorManager) monitoringLoop() {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.collectStats()
		}
	}
}

// collectStats 收集所有被监控进程的统计信息
func (m *ProcessMonitorManager) collectStats() {
	m.mu.RLock()
	processes := make(map[int]string)
	for pid, name := range m.monitoredProcesses {
		processes[pid] = name
	}
	config := m.config
	m.mu.RUnlock()

	for pid, name := range processes {
		stats, err := getProcessStats(pid)
		if err != nil {
			// 进程可能已经退出，从监控列表中移除
			m.mu.Lock()
			delete(m.monitoredProcesses, pid)
			delete(m.statsHistory, pid)
			m.mu.Unlock()
			continue
		}

		stats.Name = name
		stats.Timestamp = time.Now()

		m.mu.Lock()
		history := m.statsHistory[pid]
		history = append(history, *stats)

		// 保持历史记录不超过配置的大小
		if len(history) > config.HistorySize {
			history = history[len(history)-config.HistorySize:]
		}
		m.statsHistory[pid] = history
		m.mu.Unlock()
	}
}
