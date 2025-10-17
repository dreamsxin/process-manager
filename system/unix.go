//go:build !windows

package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dreamsxin/process-manager/types"
)

// collectStats 收集Unix系统统计信息
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

	// 获取系统负载
	load1, load5, load15, err := sm.getLoadAverage()
	if err != nil {
		// 负载信息不是必须的，忽略错误
		stats.Load1 = 0
		stats.Load5 = 0
		stats.Load15 = 0
	} else {
		stats.Load1 = load1
		stats.Load5 = load5
		stats.Load15 = load15
	}

	return stats, nil
}

// getCPUPercent 获取CPU使用率
func (sm *SystemMonitor) getCPUPercent() (float64, error) {
	// 读取/proc/stat获取CPU信息
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return 0, fmt.Errorf("invalid cpu line")
			}

			// 解析CPU时间
			user, _ := strconv.ParseUint(fields[1], 10, 64)
			nice, _ := strconv.ParseUint(fields[2], 10, 64)
			system, _ := strconv.ParseUint(fields[3], 10, 64)
			idle, _ := strconv.ParseUint(fields[4], 10, 64)
			iowait, _ := strconv.ParseUint(fields[5], 10, 64)
			irq, _ := strconv.ParseUint(fields[6], 10, 64)
			softirq, _ := strconv.ParseUint(fields[7], 10, 64)

			// 计算总CPU时间
			total := user + nice + system + idle + iowait + irq + softirq
			idleTotal := idle + iowait

			// 如果是第一次调用，保存基准值
			if lastCPUTotal == 0 {
				lastCPUTotal = total
				lastCPUIdle = idleTotal
				return 0, nil
			}

			// 计算CPU使用率
			totalDiff := total - lastCPUTotal
			idleDiff := idleTotal - lastCPUIdle

			// 更新上次的值
			lastCPUTotal = total
			lastCPUIdle = idleTotal

			if totalDiff == 0 {
				return 0, nil
			}

			cpuUsage := (1.0 - float64(idleDiff)/float64(totalDiff)) * 100.0
			return cpuUsage, nil
		}
	}

	return 0, fmt.Errorf("cpu line not found in /proc/stat")
}

// getMemoryUsage 获取内存使用情况
func (sm *SystemMonitor) getMemoryUsage() (float64, uint64, uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			memTotal, _ = strconv.ParseUint(fields[1], 10, 64)
			memTotal *= 1024 // 转换为字节
		case "MemAvailable:":
			memAvailable, _ = strconv.ParseUint(fields[1], 10, 64)
			memAvailable *= 1024 // 转换为字节
		}
	}

	if memTotal == 0 {
		return 0, 0, 0, fmt.Errorf("failed to get memory information")
	}

	memUsed := memTotal - memAvailable
	memoryPercent := (float64(memUsed) / float64(memTotal)) * 100

	return memoryPercent, memUsed, memTotal, nil
}

// getDiskUsage 获取磁盘使用情况
func (sm *SystemMonitor) getDiskUsage() (float64, uint64, uint64, error) {
	// 使用df命令获取根分区使用情况
	cmd := exec.Command("df", "/")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, fmt.Errorf("invalid df output")
	}

	// 解析第二行（数据行）
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return 0, 0, 0, fmt.Errorf("invalid df data format")
	}

	totalBlocks, _ := strconv.ParseUint(fields[1], 10, 64)
	usedBlocks, _ := strconv.ParseUint(fields[2], 10, 64)
	// availableBlocks, _ := strconv.ParseUint(fields[3], 10, 64)

	// 转换为字节（假设块大小为1KB）
	totalBytes := totalBlocks * 1024
	usedBytes := usedBlocks * 1024

	diskPercent := (float64(usedBytes) / float64(totalBytes)) * 100

	return diskPercent, usedBytes, totalBytes, nil
}

// getLoadAverage 获取系统负载
func (sm *SystemMonitor) getLoadAverage() (float64, float64, float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return load1, load5, load15, nil
}

// 添加这些全局变量用于CPU计算
var (
	lastCPUTotal uint64
	lastCPUIdle  uint64
)
