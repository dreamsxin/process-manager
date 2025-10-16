package types

import (
	"os/exec"
	"time"
)

// ProcessInfo contains information about a managed process
type ProcessInfo struct {
	UUID         string
	Cmd          *exec.Cmd
	Name         string
	Args         []string
	PID          int
	Running      bool
	Restart      bool
	StartTime    time.Time
	EndTime      time.Time
	RestartCount int
}

// Status returns the current status of the process as a string
func (p *ProcessInfo) Status() string {
	if p.Running {
		return "running"
	}
	return "stopped"
}

// Uptime returns the duration the process has been running
func (p *ProcessInfo) Uptime() time.Duration {
	if p.Running {
		return time.Since(p.StartTime)
	}
	if !p.EndTime.IsZero() {
		return p.EndTime.Sub(p.StartTime)
	}
	return 0
}

// IsActive returns true if the process is currently running
func (p *ProcessInfo) IsActive() bool {
	return p.Running
}
