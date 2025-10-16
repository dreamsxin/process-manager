//go:build !windows

package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dreamsxin/process-manager/types"
)

// getProcessStats 获取Unix进程统计信息
func getProcessStats(pid int) (*types.ProcessStats, error) {
	// 检查进程是否存在
	if !isProcessRunning(pid) {
		return nil, fmt.Errorf("process %d does not exist", pid)
	}

	// 获取进程状态信息
	stat, err := getProcessStat(pid)
	if err != nil {
		return nil, err
	}

	// 获取进程内存信息
	memoryInfo, err := getProcessMemoryInfo(pid)
	if err != nil {
		return nil, err
	}

	// 获取进程CPU使用率
	cpuPercent, err := getProcessCPUPercent(pid)
	if err != nil {
		cpuPercent = 0
	}

	// 获取内存使用百分比
	memoryPercent, err := getMemoryPercent(memoryInfo.rss)
	if err != nil {
		memoryPercent = 0
	}

	return &types.ProcessStats{
		PID:           pid,
		Name:          stat.name,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		MemoryBytes:   memoryInfo.rss,
		CreateTime:    stat.startTime,
		Timestamp:     time.Now(),
	}, nil
}

// processStat 进程状态信息
type processStat struct {
	pid       int
	name      string
	state     string
	ppid      int
	utime     uint64
	stime     uint64
	startTime time.Time
}

// processMemoryInfo 进程内存信息
type processMemoryInfo struct {
	rss   uint64 // 驻留集大小
	vsize uint64 // 虚拟内存大小
}

// cpuUsage 用于CPU使用率计算
type cpuUsage struct {
	lastTime  time.Time
	lastUTime uint64
	lastSTime uint64
}

var cpuUsageMap = make(map[int]*cpuUsage)

// getProcessStat 从/proc文件系统读取进程状态
func getProcessStat(pid int) (*processStat, error) {
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statFile)
	if err != nil {
		return nil, err
	}

	// 解析stat文件内容
	content := string(data)
	// 找到第一个和最后一个括号来提取进程名
	firstParen := strings.IndexRune(content, '(')
	lastParen := strings.LastIndex(content, ")")
	if firstParen == -1 || lastParen == -1 {
		return nil, fmt.Errorf("invalid stat format for PID %d", pid)
	}

	name := content[firstParen+1 : lastParen]
	rest := strings.Fields(content[lastParen+2:])

	if len(rest) < 20 {
		return nil, fmt.Errorf("invalid stat format for PID %d", pid)
	}

	// 解析字段
	state := rest[0]
	ppid, _ := strconv.Atoi(rest[1])
	utime, _ := strconv.ParseUint(rest[11], 10, 64)
	stime, _ := strconv.ParseUint(rest[12], 10, 64)

	// 计算启动时间
	startTime, err := getProcessStartTime(pid, rest[19])
	if err != nil {
		startTime = time.Now()
	}

	return &processStat{
		pid:       pid,
		name:      name,
		state:     state,
		ppid:      ppid,
		utime:     utime,
		stime:     stime,
		startTime: startTime,
	}, nil
}

// getProcessMemoryInfo 获取进程内存信息
func getProcessMemoryInfo(pid int) (*processMemoryInfo, error) {
	statmFile := fmt.Sprintf("/proc/%d/statm", pid)
	data, err := os.ReadFile(statmFile)
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid statm format for PID %d", pid)
	}

	// 获取页面大小
	pageSize := uint64(os.Getpagesize())

	// 解析字段
	vsize, _ := strconv.ParseUint(fields[0], 10, 64)
	rss, _ := strconv.ParseUint(fields[1], 10, 64)

	// 转换为字节
	vsize *= pageSize
	rss *= pageSize

	return &processMemoryInfo{
		rss:   rss,
		vsize: vsize,
	}, nil
}

// getProcessCPUPercent 计算进程CPU使用率
func getProcessCPUPercent(pid int) (float64, error) {
	stat, err := getProcessStat(pid)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	totalTime := stat.utime + stat.stime

	// 检查是否有上一次的记录
	usage, exists := cpuUsageMap[pid]
	if !exists {
		// 第一次采样，创建记录
		cpuUsageMap[pid] = &cpuUsage{
			lastTime:  now,
			lastUTime: stat.utime,
			lastSTime: stat.stime,
		}
		return 0, nil
	}

	// 计算时间差
	timeDiff := now.Sub(usage.lastTime).Seconds()
	if timeDiff <= 0 {
		return 0, nil
	}

	// 计算CPU时间差
	cpuTimeDiff := float64(totalTime - (usage.lastUTime + usage.lastSTime))

	// 计算CPU使用率百分比
	// 注意：这里需要知道时钟频率，通常为100
	clockTicks := 100.0
	cpuPercent := (cpuTimeDiff / clockTicks) / timeDiff * 100

	// 更新记录
	usage.lastTime = now
	usage.lastUTime = stat.utime
	usage.lastSTime = stat.stime

	// 限制在0-100之间
	if cpuPercent < 0 {
		cpuPercent = 0
	}
	if cpuPercent > 100 {
		cpuPercent = 100
	}

	return cpuPercent, nil
}

// getProcessStartTime 获取进程启动时间
func getProcessStartTime(pid int, startTimeTicks string) (time.Time, error) {
	// 读取系统启动时间
	bootTime, err := getSystemBootTime()
	if err != nil {
		return time.Time{}, err
	}

	// 解析进程启动时间ticks
	ticks, err := strconv.ParseUint(startTimeTicks, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	// 获取时钟频率（通常为100）
	clockTicks := uint64(100)

	// 计算启动时间
	startTime := bootTime.Add(time.Duration(ticks) * time.Second / time.Duration(clockTicks))
	return startTime, nil
}

// getSystemBootTime 获取系统启动时间
func getSystemBootTime() (time.Time, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				timestamp, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return time.Time{}, err
				}
				return time.Unix(timestamp, 0), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("btime not found in /proc/stat")
}

// getSystemUptime 获取系统运行时间
func getSystemUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uptime, nil
}

// getPIDsByName 根据进程名获取PID列表
func getPIDsByName(name string) ([]int, []string, error) {
	var pids []int
	var names []string

	// 遍历/proc目录
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// 读取进程状态
		stat, err := getProcessStat(pid)
		if err != nil {
			continue
		}

		if stat.name == name {
			pids = append(pids, pid)
			names = append(names, stat.name)
		}
	}

	return pids, names, nil
}

// getMemoryPercent 获取内存使用百分比
func getMemoryPercent(rss uint64) (float64, error) {
	// 读取系统内存信息
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	var totalMemory uint64
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				totalMemory = kb * 1024 // 转换为字节
				break
			}
		}
	}

	if totalMemory == 0 {
		return 0, fmt.Errorf("failed to get total memory")
	}

	return (float64(rss) / float64(totalMemory)) * 100, nil
}

// isProcessRunning 检查进程是否在运行
func isProcessRunning(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

// getProcessName 获取进程名
func getProcessName(pid int) (string, error) {
	stat, err := getProcessStat(pid)
	if err != nil {
		return "", err
	}
	return stat.name, nil
}
