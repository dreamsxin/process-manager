package manager

import (
	"fmt"
	"sync"

	"github.com/dreamsxin/process-manager/monitor"
	"github.com/dreamsxin/process-manager/types"
)

// ProcessManagerWithMonitor 带监控功能的进程管理器
type ProcessManagerWithMonitor struct {
	*ProcessManager
	monitorManager *monitor.ProcessMonitorManager
	mu             sync.RWMutex
}

// NewProcessManagerWithMonitor 创建带监控功能的进程管理器
func NewProcessManagerWithMonitor() *ProcessManagerWithMonitor {
	pm := &ProcessManagerWithMonitor{
		ProcessManager: NewProcessManager(),
		monitorManager: monitor.NewProcessMonitorManager(),
	}

	// 启动监控
	go pm.monitorManager.Start()

	return pm
}

// StartProcess 启动进程并添加到监控
func (pm *ProcessManagerWithMonitor) StartProcess(name string, args []string, restart bool) (string, error) {
	uuid, err := pm.ProcessManager.StartProcess(name, args, restart)
	if err != nil {
		return "", err
	}

	// 获取进程信息并添加到监控
	if processInfo, exists := pm.GetProcess(uuid); exists {
		pm.monitorManager.AddProcess(processInfo.PID, processInfo.Name)
	}

	return uuid, nil
}

// StopProcess 停止进程并从监控移除
func (pm *ProcessManagerWithMonitor) StopProcess(uuid string) error {
	processInfo, exists := pm.GetProcess(uuid)
	if exists {
		// 从监控中移除
		pm.monitorManager.RemoveProcess(processInfo.PID)
	}

	return pm.ProcessManager.StopProcess(uuid)
}

// StopAll 停止所有进程并清理监控
func (pm *ProcessManagerWithMonitor) StopAll() {
	// 先停止监控
	pm.monitorManager.Stop()

	// 然后停止所有进程
	pm.ProcessManager.StopAll()
}

// Shutdown 关闭进程管理器和监控
func (pm *ProcessManagerWithMonitor) Shutdown() {
	pm.StopAll()
	pm.ProcessManager.Shutdown()
}

// 监控相关方法

// GetProcessStats 获取进程统计信息
func (pm *ProcessManagerWithMonitor) GetProcessStats(pid int) (*types.ProcessStats, error) {
	return pm.monitorManager.GetProcessStats(pid)
}

// GetProcessStatsByName 按进程名获取统计信息
func (pm *ProcessManagerWithMonitor) GetProcessStatsByName(name string) ([]types.ProcessStats, error) {
	return pm.monitorManager.GetProcessStatsByName(name)
}

// GetProcessStatsByUUID 按UUID获取进程统计信息
func (pm *ProcessManagerWithMonitor) GetProcessStatsByUUID(uuid string) (*types.ProcessStats, error) {
	processInfo, exists := pm.GetProcess(uuid)
	if !exists {
		return nil, fmt.Errorf("process with UUID %s not found", uuid)
	}

	return pm.monitorManager.GetProcessStats(processInfo.PID)
}

// GetAllMonitoredStats 获取所有被监控进程的统计信息
func (pm *ProcessManagerWithMonitor) GetAllMonitoredStats() ([]types.ProcessStats, error) {
	return pm.monitorManager.GetAllStats()
}

// GetProcessHistory 获取进程历史统计
func (pm *ProcessManagerWithMonitor) GetProcessHistory(pid int, count int) ([]types.ProcessStats, error) {
	return pm.monitorManager.GetProcessHistory(pid, count)
}

// GetProcessHistoryByUUID 按UUID获取进程历史统计
func (pm *ProcessManagerWithMonitor) GetProcessHistoryByUUID(uuid string, count int) ([]types.ProcessStats, error) {
	processInfo, exists := pm.GetProcess(uuid)
	if !exists {
		return nil, fmt.Errorf("process with UUID %s not found", uuid)
	}

	return pm.monitorManager.GetProcessHistory(processInfo.PID, count)
}

// AddProcessToMonitor 添加进程到监控
func (pm *ProcessManagerWithMonitor) AddProcessToMonitor(pid int, name string) error {
	return pm.monitorManager.AddProcess(pid, name)
}

// RemoveProcessFromMonitor 从监控移除进程
func (pm *ProcessManagerWithMonitor) RemoveProcessFromMonitor(pid int) error {
	return pm.monitorManager.RemoveProcess(pid)
}

// GetMonitorConfig 获取监控配置
func (pm *ProcessManagerWithMonitor) GetMonitorConfig() types.MonitorConfig {
	return pm.monitorManager.GetConfig()
}

// UpdateMonitorConfig 更新监控配置
func (pm *ProcessManagerWithMonitor) UpdateMonitorConfig(config types.MonitorConfig) error {
	return pm.monitorManager.UpdateConfig(config)
}

// GetMonitoredProcesses 获取被监控的进程列表
func (pm *ProcessManagerWithMonitor) GetMonitoredProcesses() map[int]string {
	return pm.monitorManager.GetMonitoredProcesses()
}

// MonitorProcessByName 按进程名监控进程
func (pm *ProcessManagerWithMonitor) MonitorProcessByName(name string) error {
	pids, err := pm.monitorManager.GetProcessStatsByName(name)
	if err != nil {
		return err
	}

	for _, stats := range pids {
		pm.monitorManager.AddProcess(stats.PID, stats.Name)
	}

	return nil
}
