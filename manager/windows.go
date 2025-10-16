//go:build windows

package manager

import (
	"fmt"
	"os/exec"
)

// killProcessWindows terminates a process and its children on Windows
func (pm *ProcessManager) killProcessWindows(cmd *exec.Cmd) error {
	// Use taskkill to terminate the entire process tree
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
	return killCmd.Run()
}
