//go:build !windows

package manager

import (
	"os/exec"
	"syscall"
	"time"
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

	// First try SIGTERM for graceful shutdown
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	if err != nil {
		// If process doesn't exist, that's fine
		if err == syscall.ESRCH {
			return nil
		}
	}

	// Wait a bit for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Check if process is still running
	if pm.isProcessRunning(cmd.Process.Pid) {
		// Force kill with SIGKILL
		err = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err != nil && err != syscall.ESRCH {
			return err
		}
	}

	return nil
}

// isProcessRunning 检查进程是否仍在运行
func (pm *ProcessManager) isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	err := syscall.Kill(pid, 0)
	return err == nil
}
