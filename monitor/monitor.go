package monitor

import (
	"github.com/dreamsxin/process-manager/types"
)

// Monitor 监控器接口
type Monitor interface {
	// 启动监控
	Start() error

	// 停止监控
	Stop() error

	// 获取进程统计信息
	GetProcessStats(pid int) (*types.ProcessStats, error)

	// 按进程名获取统计信息
	GetProcessStatsByName(name string) ([]types.ProcessStats, error)

	// 获取所有被监控进程的统计信息
	GetAllStats() ([]types.ProcessStats, error)

	// 获取进程历史统计
	GetProcessHistory(pid int, count int) ([]types.ProcessStats, error)

	// 添加进程到监控列表
	AddProcess(pid int, name string) error

	// 从监控列表移除进程
	RemoveProcess(pid int) error

	// 获取监控配置
	GetConfig() types.MonitorConfig

	// 更新监控配置
	UpdateConfig(config types.MonitorConfig) error
}
