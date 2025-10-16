package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/dreamsxin/process-manager/manager"
)

func main() {
	// 创建带监控的进程管理器
	pm := manager.NewProcessManagerWithMonitor()
	defer pm.Shutdown()

	fmt.Println("Starting process manager with monitoring...")

	// 启动一些示例进程
	var processUUIDs []string

	if runtime.GOOS == "windows" {
		// Windows示例
		uuid, err := pm.StartProcess("ping", []string{"127.0.0.1", "-n", "10"}, true)
		if err != nil {
			log.Printf("Error starting ping: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started ping process: %s\n", uuid)
		}

		// 启动一个长期运行的进程
		uuid, err = pm.StartProcess("timeout", []string{"30"}, false)
		if err != nil {
			log.Printf("Error starting timeout: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started timeout process: %s\n", uuid)
		}
	} else {
		// Unix示例
		uuid, err := pm.StartProcess("ping", []string{"127.0.0.1", "-c", "10"}, true)
		if err != nil {
			log.Printf("Error starting ping: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started ping process: %s\n", uuid)
		}

		// 启动一个长期运行的进程
		uuid, err = pm.StartProcess("sleep", []string{"30"}, false)
		if err != nil {
			log.Printf("Error starting sleep: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started sleep process: %s\n", uuid)
		}
	}

	// 显示监控配置
	config := pm.GetMonitorConfig()
	fmt.Printf("\nMonitor Config: Interval=%v, HistorySize=%d\n", config.Interval, config.HistorySize)

	// 显示被监控的进程
	fmt.Printf("\nMonitored Processes:\n")
	monitored := pm.GetMonitoredProcesses()
	for pid, name := range monitored {
		fmt.Printf("  PID: %d, Name: %s\n", pid, name)
	}

	// 监控循环
	fmt.Printf("\nStarting monitoring loop (5 iterations)...\n")
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		fmt.Printf("\n=== Monitoring Report #%d ===\n", i+1)

		// 获取所有被监控进程的统计信息
		stats, err := pm.GetAllMonitoredStats()
		if err != nil {
			log.Printf("Error getting stats: %v", err)
			continue
		}

		// 显示统计信息
		if len(stats) == 0 {
			fmt.Println("No processes being monitored")
			continue
		}

		for _, stat := range stats {
			fmt.Printf("Process: %s (PID: %d)\n", stat.Name, stat.PID)
			fmt.Printf("  CPU: %.2f%%, Memory: %.2f%% (%s)\n",
				stat.CPUPercent, stat.MemoryPercent, formatBytes(stat.MemoryBytes))
			fmt.Printf("  Running: %v\n", time.Since(stat.CreateTime).Round(time.Second))
			fmt.Printf("  Last Update: %v\n", stat.Timestamp.Format("15:04:05"))
		}

		// 演示按进程名监控
		if i == 0 {
			fmt.Printf("\nAdding system processes to monitoring...\n")
			// 尝试监控一些系统进程
			systemProcesses := []string{"bash", "zsh", "fish", "cmd", "powershell"}
			for _, procName := range systemProcesses {
				stats, err := pm.GetProcessStatsByName(procName)
				if err == nil && len(stats) > 0 {
					fmt.Printf("Found %d %s processes, adding to monitoring...\n", len(stats), procName)
					for _, stat := range stats {
						pm.AddProcessToMonitor(stat.PID, stat.Name)
					}
				}
			}
		}
	}

	// 显示详细的历史数据
	if len(processUUIDs) > 0 {
		fmt.Printf("\n=== Process History (last 3 samples) ===\n")
		history, err := pm.GetProcessHistoryByUUID(processUUIDs[0], 3)
		if err == nil {
			for i, stat := range history {
				fmt.Printf("Sample #%d: CPU=%.2f%%, Memory=%.2f%%, Time=%v\n",
					i+1, stat.CPUPercent, stat.MemoryPercent, stat.Timestamp.Format("15:04:05"))
			}
		}
	}

	// 演示单个进程监控
	if len(processUUIDs) > 0 {
		fmt.Printf("\n=== Single Process Monitoring ===\n")
		stats, err := pm.GetProcessStatsByUUID(processUUIDs[0])
		if err == nil {
			fmt.Printf("Process: %s (PID: %d)\n", stats.Name, stats.PID)
			fmt.Printf("Current CPU: %.2f%%\n", stats.CPUPercent)
			fmt.Printf("Current Memory: %.2f%% (%s)\n", stats.MemoryPercent, formatBytes(stats.MemoryBytes))
		}
	}

	fmt.Println("\nMonitoring demo completed.")
}

// formatBytes 格式化字节大小为人类可读的格式
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
