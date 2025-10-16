package manager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dreamsxin/process-manager/types"
	"github.com/dreamsxin/process-manager/util"
)

// ProcessManager manages multiple processes with UUID-based identification
type ProcessManager struct {
	processes sync.Map // key: UUID, value: *types.ProcessInfo
	mu        sync.RWMutex
	shutdown  chan struct{}
	wg        sync.WaitGroup
}

// NewProcessManager creates a new ProcessManager instance
func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		shutdown: make(chan struct{}),
	}

	// Setup signal handling for graceful shutdown
	pm.setupSignalHandling()
	return pm
}

// StartProcess starts a new process and returns its UUID
func (pm *ProcessManager) StartProcess(name string, args []string, restart bool) (string, error) {
	uuid := util.GenerateUUID()

	cmd, err := pm.createCommand(name, args)
	if err != nil {
		return "", fmt.Errorf("failed to create command: %v", err)
	}

	processInfo := &types.ProcessInfo{
		UUID:         uuid,
		Cmd:          cmd,
		Name:         name,
		Args:         args,
		Running:      false,
		Restart:      restart,
		StartTime:    time.Now(),
		RestartCount: 0,
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start process: %v", err)
	}

	processInfo.Running = true
	processInfo.PID = cmd.Process.Pid
	pm.processes.Store(uuid, processInfo)

	// Monitor process in background
	pm.wg.Add(1)
	go pm.monitorProcess(uuid, processInfo)

	fmt.Printf("Started process: %s (UUID: %s, PID: %d)\n", name, uuid, cmd.Process.Pid)
	return uuid, nil
}

// RestartProcess restarts a process by UUID and returns the new UUID
func (pm *ProcessManager) RestartProcess(uuid string) (string, error) {
	value, exists := pm.processes.Load(uuid)
	if !exists {
		return "", fmt.Errorf("process with UUID %s not found", uuid)
	}

	processInfo := value.(*types.ProcessInfo)

	// Stop the current process if it's running
	if processInfo.Running {
		if err := pm.killProcess(processInfo.Cmd); err != nil {
			return "", fmt.Errorf("failed to stop process for restart: %v", err)
		}
		// Brief pause to ensure process is fully terminated
		time.Sleep(100 * time.Millisecond)
	}

	// Remove old process record
	pm.processes.Delete(uuid)

	// Start new process with same configuration
	newUUID, err := pm.StartProcess(processInfo.Name, processInfo.Args, processInfo.Restart)
	if err != nil {
		return "", fmt.Errorf("failed to restart process: %v", err)
	}

	// Update restart count in new process info
	if newValue, exists := pm.processes.Load(newUUID); exists {
		newProcessInfo := newValue.(*types.ProcessInfo)
		newProcessInfo.RestartCount = processInfo.RestartCount + 1
	}

	fmt.Printf("Restarted process: %s (Old UUID: %s, New UUID: %s)\n",
		processInfo.Name, uuid, newUUID)
	return newUUID, nil
}

// StopProcess stops a specific process by UUID
func (pm *ProcessManager) StopProcess(uuid string) error {
	value, exists := pm.processes.Load(uuid)
	if !exists {
		return fmt.Errorf("process with UUID %s not found", uuid)
	}

	processInfo := value.(*types.ProcessInfo)
	processInfo.Restart = false // Disable auto-restart

	if processInfo.Running {
		if err := pm.killProcess(processInfo.Cmd); err != nil {
			// 检查进程是否已经退出
			if pm.isProcessRunning(processInfo.PID) {
				return fmt.Errorf("failed to stop process: %v", err)
			}
			// 如果进程已经退出，我们认为终止成功
		}
	}

	pm.processes.Delete(uuid)
	fmt.Printf("Stopped process: %s (UUID: %s)\n", processInfo.Name, uuid)
	return nil
}

// StopAll stops all managed processes
func (pm *ProcessManager) StopAll() {
	var wg sync.WaitGroup

	pm.processes.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(uuid string, processInfo *types.ProcessInfo) {
			defer wg.Done()
			processInfo.Restart = false
			if processInfo.Running {
				// 尝试终止进程，但忽略错误
				pm.killProcess(processInfo.Cmd)
			}
			fmt.Printf("Stopped process: %s (UUID: %s)\n", processInfo.Name, uuid)
		}(key.(string), value.(*types.ProcessInfo))
		return true
	})

	wg.Wait()
	pm.processes = sync.Map{} // Clear the map
	fmt.Println("All processes stopped")
}

// GetProcess retrieves process information by UUID
func (pm *ProcessManager) GetProcess(uuid string) (*types.ProcessInfo, bool) {
	value, exists := pm.processes.Load(uuid)
	if !exists {
		return nil, false
	}
	return value.(*types.ProcessInfo), true
}

// ListProcesses returns a list of all managed processes
func (pm *ProcessManager) ListProcesses() []*types.ProcessInfo {
	var processes []*types.ProcessInfo

	pm.processes.Range(func(key, value interface{}) bool {
		processes = append(processes, value.(*types.ProcessInfo))
		return true
	})

	return processes
}

// WaitForProcess waits for a specific process to complete with timeout
func (pm *ProcessManager) WaitForProcess(uuid string, timeout time.Duration) error {
	value, exists := pm.processes.Load(uuid)
	if !exists {
		return fmt.Errorf("process with UUID %s not found", uuid)
	}

	processInfo := value.(*types.ProcessInfo)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		if processInfo.Cmd.Process != nil {
			_, err := processInfo.Cmd.Process.Wait()
			done <- err
		} else {
			done <- nil
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait timeout for process %s", uuid)
	case err := <-done:
		return err
	}
}

// Shutdown gracefully shuts down the process manager and all processes
func (pm *ProcessManager) Shutdown() {
	fmt.Println("Shutting down process manager...")
	close(pm.shutdown)
	pm.StopAll()
	pm.wg.Wait()
	fmt.Println("Process manager shutdown complete")
}

// setupSignalHandling configures OS signal handling for graceful shutdown
func (pm *ProcessManager) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal")
		pm.Shutdown()
		os.Exit(0)
	}()
}

// monitorProcess monitors a process and handles auto-restart if enabled
func (pm *ProcessManager) monitorProcess(uuid string, processInfo *types.ProcessInfo) {
	defer pm.wg.Done()

	err := processInfo.Cmd.Wait()
	if err != nil {
		fmt.Printf("Process %s (UUID: %s) exited with error: %v\n", processInfo.Name, uuid, err)
	}

	pm.mu.Lock()
	processInfo.Running = false
	processInfo.EndTime = time.Now()
	pm.mu.Unlock()

	// Check if we should restart
	select {
	case <-pm.shutdown:
		// Manager is shutting down, don't restart
		pm.processes.Delete(uuid)
		return
	default:
		// Continue with restart logic
	}

	if processInfo.Restart {
		processInfo.RestartCount++
		fmt.Printf("Auto-restarting process: %s (UUID: %s, Restart count: %d)\n",
			processInfo.Name, uuid, processInfo.RestartCount)

		time.Sleep(2 * time.Second)

		// Check if process is still in manager and restart is still enabled
		if currentValue, exists := pm.processes.Load(uuid); exists {
			currentInfo := currentValue.(*types.ProcessInfo)
			if currentInfo.Restart {
				pm.RestartProcess(uuid)
				return
			}
		}
	}

	// Process ended and won't restart, remove from manager
	pm.processes.Delete(uuid)
}

// killProcess is a platform-agnostic method that delegates to platform-specific implementations
func (pm *ProcessManager) killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return pm.killProcessPlatform(cmd)
}
