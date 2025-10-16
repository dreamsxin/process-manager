//go:build !windows

package manager

import (
	"os/exec"
	"syscall"
)

// killProcessUnix terminates a process and its children on Unix systems
func (pm *ProcessManager) killProcessUnix(cmd *exec.Cmd) error {
	// Use process group to kill all child processes
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
