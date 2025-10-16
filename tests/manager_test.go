package tests

import (
	"runtime"
	"testing"
	"time"

	"github.com/dreamsxin/process-manager/manager"
)

func TestProcessManagerLifecycle(t *testing.T) {
	pm := manager.NewProcessManager()
	defer pm.Shutdown()

	var testCommand string
	var testArgs []string

	if runtime.GOOS == "windows" {
		testCommand = "cmd"
		testArgs = []string{"/c", "echo", "test"}
	} else {
		testCommand = "echo"
		testArgs = []string{"test"}
	}

	// Test starting a process
	uuid, err := pm.StartProcess(testCommand, testArgs, false)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Verify process is in list
	processes := pm.ListProcesses()
	if len(processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(processes))
	}

	// Test getting process info
	process, exists := pm.GetProcess(uuid)
	if !exists {
		t.Error("Process not found by UUID")
	}

	if process.Name != testCommand {
		t.Errorf("Expected process name %s, got %s", testCommand, process.Name)
	}

	// Wait for process to complete
	time.Sleep(1 * time.Second)

	// Process should be removed after completion (since restart=false)
	processes = pm.ListProcesses()
	if len(processes) != 0 {
		t.Errorf("Expected 0 processes after completion, got %d", len(processes))
	}
}

func TestProcessRestart(t *testing.T) {
	pm := manager.NewProcessManager()
	defer pm.Shutdown()

	var testCommand string
	var testArgs []string

	if runtime.GOOS == "windows" {
		testCommand = "cmd"
		testArgs = []string{"/c", "timeout", "1"}
	} else {
		testCommand = "sleep"
		testArgs = []string{"1"}
	}

	// Start a process with auto-restart enabled
	uuid, err := pm.StartProcess(testCommand, testArgs, true)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait for process to complete and restart
	time.Sleep(3 * time.Second)

	// Process should still be in the list due to auto-restart
	processes := pm.ListProcesses()
	if len(processes) != 1 {
		t.Errorf("Expected 1 process after auto-restart, got %d", len(processes))
	}

	// Check that it's a different process (new UUID after restart)
	newProcess := processes[0]
	if newProcess.UUID == uuid {
		t.Error("Expected new UUID after restart, but got same UUID")
	}

	if newProcess.RestartCount < 1 {
		t.Errorf("Expected restart count >= 1, got %d", newProcess.RestartCount)
	}
}

func TestProcessStop(t *testing.T) {
	pm := manager.NewProcessManager()
	defer pm.Shutdown()

	var testCommand string
	var testArgs []string

	if runtime.GOOS == "windows" {
		testCommand = "cmd"
		testArgs = []string{"/c", "timeout", "10"}
	} else {
		testCommand = "sleep"
		testArgs = []string{"10"}
	}

	// Start a long-running process
	uuid, err := pm.StartProcess(testCommand, testArgs, false)
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Verify process is running
	processes := pm.ListProcesses()
	if len(processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(processes))
	}

	// Stop the process
	err = pm.StopProcess(uuid)
	if err != nil {
		t.Fatalf("Failed to stop process: %v", err)
	}

	// Verify process is removed
	processes = pm.ListProcesses()
	if len(processes) != 0 {
		t.Errorf("Expected 0 processes after stop, got %d", len(processes))
	}
}

func TestStopAll(t *testing.T) {
	pm := manager.NewProcessManager()
	defer pm.Shutdown()

	var testCommand string
	var testArgs []string

	if runtime.GOOS == "windows" {
		testCommand = "cmd"
		testArgs = []string{"/c", "timeout", "10"}
	} else {
		testCommand = "sleep"
		testArgs = []string{"10"}
	}

	// Start multiple processes
	_, err := pm.StartProcess(testCommand, testArgs, false)
	if err != nil {
		t.Fatalf("Failed to start first process: %v", err)
	}

	_, err = pm.StartProcess(testCommand, testArgs, false)
	if err != nil {
		t.Fatalf("Failed to start second process: %v", err)
	}

	// Verify both processes are running
	processes := pm.ListProcesses()
	if len(processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(processes))
	}

	// Stop all processes
	pm.StopAll()

	// Verify no processes are running
	processes = pm.ListProcesses()
	if len(processes) != 0 {
		t.Errorf("Expected 0 processes after StopAll, got %d", len(processes))
	}
}
