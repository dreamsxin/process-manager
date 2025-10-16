//go:build !windows

package manager

import (
	"os/exec"
	"syscall"
)

// createCommand creates a Unix-specific command
func (pm *ProcessManager) createCommand(name string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create process group for Unix systems
	}
	return cmd, nil
}

// killProcessPlatform terminates a process and its children on Unix systems
func (pm *ProcessManager) killProcessPlatform(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	// Use process group to kill all child processes
	// Negative PID means kill the process group
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err != nil {
		// 如果进程已经不存在，忽略错误
		if err == syscall.ESRCH {
			return nil
		}
		return err
	}
	return nil
}

// isProcessRunning 检查进程是否仍在运行
func (pm *ProcessManager) isProcessRunning(pid int) bool {
	// 向进程发送信号0来检查是否存在
	err := syscall.Kill(pid, 0)
	return err == nil
}
