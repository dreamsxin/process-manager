//go:build windows

package manager

import (
	"fmt"
	"os/exec"
	"syscall"
)

const (
	CREATE_NEW_PROCESS_GROUP = 0x00000200
)

// createCommand creates a Windows-specific command
func (pm *ProcessManager) createCommand(name string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: CREATE_NEW_PROCESS_GROUP,
	}
	return cmd, nil
}

// killProcessPlatform terminates a process and its children on Windows
func (pm *ProcessManager) killProcessPlatform(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid

	// 方法1: 使用taskkill (最可靠的方法)
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	if err := killCmd.Run(); err == nil {
		return nil
	}

	// 方法2: 使用wmic (备用方法)
	wmicCmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "delete")
	if err := wmicCmd.Run(); err == nil {
		return nil
	}

	// 方法3: 直接使用TerminateProcess API (最底层的方法)
	return pm.terminateProcessAPI(pid)
}

// terminateProcessAPI 使用Windows API直接终止进程
func (pm *ProcessManager) terminateProcessAPI(pid int) error {
	// 定义必要的常量
	const (
		PROCESS_TERMINATE         = 0x0001
		PROCESS_QUERY_INFORMATION = 0x0400
	)

	// 打开进程句柄
	handle, err := syscall.OpenProcess(PROCESS_TERMINATE|PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		// 如果进程已经不存在，我们认为终止成功
		// 使用错误号而不是预定义的常量
		if errno, ok := err.(syscall.Errno); ok {
			// ERROR_ACCESS_DENIED = 5, ERROR_INVALID_PARAMETER = 87
			if errno == 5 || errno == 87 {
				return nil
			}
		}
		return fmt.Errorf("failed to open process %d: %v", pid, err)
	}
	defer syscall.CloseHandle(handle)

	// 终止进程
	err = syscall.TerminateProcess(handle, 1)
	if err != nil {
		// 如果进程已经终止，忽略错误
		if errno, ok := err.(syscall.Errno); ok && errno == 5 {
			return nil
		}
		return fmt.Errorf("failed to terminate process %d: %v", pid, err)
	}

	return nil
}

// isProcessRunning 检查进程是否仍在运行
func (pm *ProcessManager) isProcessRunning(pid int) bool {
	const (
		PROCESS_QUERY_INFORMATION = 0x0400
		STILL_ACTIVE              = 259
	)

	handle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}

	return exitCode == STILL_ACTIVE
}
