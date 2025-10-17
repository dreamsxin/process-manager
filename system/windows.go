//go:build windows

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/dreamsxin/process-manager/types"
)

// 定义Windows内存状态结构体
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// collectStats 收集Windows系统统计信息
func (sm *SystemMonitor) collectStats() (*types.SystemStats, error) {
	stats := &types.SystemStats{
		Timestamp: time.Now(),
	}

	// 获取CPU使用率
	cpuPercent, err := sm.getCPUPercent()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU stats: %v", err)
	}
	stats.CPUPercent = cpuPercent

	// 获取内存使用率
	memoryPercent, memoryUsed, memoryTotal, err := sm.getMemoryUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %v", err)
	}
	stats.MemoryPercent = memoryPercent
	stats.MemoryUsed = memoryUsed
	stats.MemoryTotal = memoryTotal

	// 获取磁盘使用率
	diskPercent, diskUsed, diskTotal, err := sm.getDiskUsage()
	if err != nil {
		// 磁盘信息不是必须的，忽略错误
		stats.DiskPercent = 0
		stats.DiskUsed = 0
		stats.DiskTotal = 0
	} else {
		stats.DiskPercent = diskPercent
		stats.DiskUsed = diskUsed
		stats.DiskTotal = diskTotal
	}

	// Windows没有直接的负载平均值，可以跳过或使用其他指标
	stats.Load1 = 0
	stats.Load5 = 0
	stats.Load15 = 0

	return stats, nil
}

// getCPUPercent 获取CPU使用率
func (sm *SystemMonitor) getCPUPercent() (float64, error) {
	// 使用Windows Performance Counters获取CPU使用率
	// 这里使用wmic命令作为替代方案
	cmd := exec.Command("wmic", "cpu", "get", "LoadPercentage", "/value")
	output, err := cmd.Output()
	if err != nil {
		// 如果wmic失败，尝试使用typeperf
		return sm.getCPUPercentFallback()
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "LoadPercentage=") {
			cpuStr := strings.TrimSpace(strings.TrimPrefix(line, "LoadPercentage="))
			cpuValue, err := strconv.ParseFloat(cpuStr, 64)
			if err != nil {
				return 0, err
			}
			return cpuValue, nil
		}
	}

	return 0, fmt.Errorf("failed to parse CPU usage")
}

// getCPUPercentFallback 备用的CPU使用率获取方法
func (sm *SystemMonitor) getCPUPercentFallback() (float64, error) {
	// 使用PowerShell获取CPU使用率
	cmd := exec.Command("powershell", "-Command",
		"Get-WmiObject Win32_Processor | Measure-Object -Property LoadPercentage -Average | Select-Object -ExpandProperty Average")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU usage: %v", err)
	}

	cpuStr := strings.TrimSpace(string(output))
	cpuValue, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU usage: %v", err)
	}

	return cpuValue, nil
}

// getMemoryUsage 获取内存使用情况
func (sm *SystemMonitor) getMemoryUsage() (float64, uint64, uint64, error) {
	// 使用wmic命令获取内存信息（更兼容的方法）
	cmd := exec.Command("wmic", "ComputerSystem", "get", "TotalPhysicalMemory", "/value")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total memory: %v", err)
	}

	var totalMemory uint64
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "TotalPhysicalMemory=") {
			memStr := strings.TrimSpace(strings.TrimPrefix(line, "TotalPhysicalMemory="))
			totalMemory, err = strconv.ParseUint(memStr, 10, 64)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("failed to parse total memory: %v", err)
			}
			break
		}
	}

	if totalMemory == 0 {
		return 0, 0, 0, fmt.Errorf("failed to get total memory")
	}

	// 获取可用内存
	cmd = exec.Command("wmic", "OS", "get", "FreePhysicalMemory", "/value")
	output, err = cmd.Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get free memory: %v", err)
	}

	var freeMemoryKB uint64
	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "FreePhysicalMemory=") {
			memStr := strings.TrimSpace(strings.TrimPrefix(line, "FreePhysicalMemory="))
			freeMemoryKB, err = strconv.ParseUint(memStr, 10, 64)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("failed to parse free memory: %v", err)
			}
			break
		}
	}

	// 转换为字节
	freeMemory := freeMemoryKB * 1024
	usedMemory := totalMemory - freeMemory
	memoryPercent := (float64(usedMemory) / float64(totalMemory)) * 100

	return memoryPercent, usedMemory, totalMemory, nil
}

// getMemoryUsageEx 使用Windows API获取内存使用情况（备选方案）
func (sm *SystemMonitor) getMemoryUsageEx() (float64, uint64, uint64, error) {
	// 加载kernel32.dll
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	// 准备内存状态结构体
	var memStatus memoryStatusEx
	memStatus.Length = uint32(unsafe.Sizeof(memStatus))

	// 调用GlobalMemoryStatusEx
	ret, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return 0, 0, 0, fmt.Errorf("GlobalMemoryStatusEx failed: %v", err)
	}

	totalMemory := memStatus.TotalPhys
	availableMemory := memStatus.AvailPhys
	usedMemory := totalMemory - availableMemory
	memoryPercent := float64(memStatus.MemoryLoad)

	return memoryPercent, usedMemory, totalMemory, nil
}

// getDiskUsage 获取磁盘使用情况
func (sm *SystemMonitor) getDiskUsage() (float64, uint64, uint64, error) {
	// 使用wmic获取C盘使用情况
	cmd := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'", "get", "Size,FreeSpace", "/format:value")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	var totalSpace, freeSpace uint64
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "FreeSpace=") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "FreeSpace="))
			freeSpace, _ = strconv.ParseUint(value, 10, 64)
		} else if strings.HasPrefix(line, "Size=") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Size="))
			totalSpace, _ = strconv.ParseUint(value, 10, 64)
		}
	}

	if totalSpace == 0 {
		return 0, 0, 0, fmt.Errorf("failed to get disk information")
	}

	usedSpace := totalSpace - freeSpace
	diskPercent := (float64(usedSpace) / float64(totalSpace)) * 100

	return diskPercent, usedSpace, totalSpace, nil
}

// 添加这些全局变量用于CPU计算（如果需要）
var (
	lastCPUTotal uint64
	lastCPUIdle  uint64
)
