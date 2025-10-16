//go:build windows

package monitor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dreamsxin/process-manager/types"
)

// cpuUsage 用于CPU使用率计算
type cpuUsage struct {
	lastTime  time.Time
	lastUTime uint64
	lastSTime uint64
}

var cpuUsageMap = make(map[int]*cpuUsage)

// getProcessStats 获取Windows进程统计信息
func getProcessStats(pid int) (*types.ProcessStats, error) {
	// 使用wmic获取进程信息
	name, err := getProcessName(pid)
	if err != nil {
		return nil, err
	}

	// 获取CPU使用率
	cpuPercent, err := getProcessCPUPercent(pid)
	if err != nil {
		cpuPercent = 0
	}

	// 获取内存信息
	memoryBytes, memoryPercent, err := getProcessMemoryInfo(pid)
	if err != nil {
		memoryBytes = 0
		memoryPercent = 0
	}

	return &types.ProcessStats{
		PID:           pid,
		Name:          name,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		MemoryBytes:   memoryBytes,
		CreateTime:    time.Now(), // Windows上获取精确创建时间较复杂
		Timestamp:     time.Now(),
	}, nil
}

// getProcessCPUPercent 获取进程CPU使用率
func getProcessCPUPercent(pid int) (float64, error) {
	// 使用wmic获取进程CPU时间
	cmd := exec.Command("wmic", "path", "Win32_PerfFormattedData_PerfProc_Process", "where", fmt.Sprintf("IDProcess=%d", pid), "get", "PercentProcessorTime", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PercentProcessorTime=") {
			cpuStr := strings.TrimSpace(strings.TrimPrefix(line, "PercentProcessorTime="))
			cpu, err := strconv.ParseFloat(cpuStr, 64)
			if err != nil {
				return 0, err
			}
			return cpu / float64(runtime.NumCPU()), nil
		}
	}

	return 0, fmt.Errorf("CPU usage not found for PID %d", pid)
}

// getProcessMemoryInfo 获取进程内存信息
func getProcessMemoryInfo(pid int) (uint64, float64, error) {
	// 使用wmic获取进程内存信息
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "WorkingSetSize", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	var memoryBytes uint64
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "WorkingSetSize=") {
			memStr := strings.TrimSpace(strings.TrimPrefix(line, "WorkingSetSize="))
			memoryBytes, err = strconv.ParseUint(memStr, 10, 64)
			if err != nil {
				return 0, 0, err
			}
			break
		}
	}

	// 获取系统总内存来计算百分比
	totalMemory, err := getTotalMemory()
	if err != nil {
		return memoryBytes, 0, nil
	}

	memoryPercent := (float64(memoryBytes) / float64(totalMemory)) * 100
	return memoryBytes, memoryPercent, nil
}

// getProcessName 获取进程名
func getProcessName(pid int) (string, error) {
	// 使用wmic获取进程名
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "Name", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Name=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Name=")), nil
		}
	}

	return "", fmt.Errorf("process name not found for PID %d", pid)
}

// getPIDsByName 根据进程名获取PID列表
func getPIDsByName(name string) ([]int, []string, error) {
	// 使用wmic根据进程名获取PID
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("Name='%s'", name), "get", "ProcessId,Name", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	var pids []int
	var names []string

	lines := strings.Split(string(output), "\n")
	var currentPID int
	var currentName string

	for _, line := range lines {
		if strings.HasPrefix(line, "ProcessId=") {
			pidStr := strings.TrimSpace(strings.TrimPrefix(line, "ProcessId="))
			currentPID, _ = strconv.Atoi(pidStr)
		} else if strings.HasPrefix(line, "Name=") {
			currentName = strings.TrimSpace(strings.TrimPrefix(line, "Name="))

			// 当收集到完整的进程信息时，添加到结果
			if currentPID > 0 && currentName != "" {
				pids = append(pids, currentPID)
				names = append(names, currentName)
				currentPID = 0
				currentName = ""
			}
		}
	}

	return pids, names, nil
}

// getTotalMemory 获取系统总内存
func getTotalMemory() (uint64, error) {
	cmd := exec.Command("wmic", "computersystem", "get", "TotalPhysicalMemory", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "TotalPhysicalMemory=") {
			memStr := strings.TrimSpace(strings.TrimPrefix(line, "TotalPhysicalMemory="))
			return strconv.ParseUint(memStr, 10, 64)
		}
	}

	return 0, fmt.Errorf("total memory not found")
}
