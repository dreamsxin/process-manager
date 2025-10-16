package manager

import (
	"os/exec"
	"runtime"
	"syscall"
)

// createCommand creates a platform-specific command with appropriate attributes
func (pm *ProcessManager) createCommand(name string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)

	// Set platform-specific process attributes
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create process group for Unix systems
		}
	}

	return cmd, nil
}

// killProcess terminates a process in a cross-platform way
func (pm *ProcessManager) killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		return pm.killProcessWindows(cmd)
	} else {
		return pm.killProcessUnix(cmd)
	}
}
