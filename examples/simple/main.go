package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/dreamsxin/process-manager/manager"
)

func main() {
	// Create a new process manager
	pm := manager.NewProcessManager()
	defer pm.Shutdown()

	var processUUIDs []string

	fmt.Println("Starting process manager demo...")

	// Start some example processes based on the OS
	if runtime.GOOS == "windows" {
		// Windows examples
		uuid, err := pm.StartProcess("ping", []string{"127.0.0.1", "-n", "10"}, true)
		if err != nil {
			log.Printf("Error starting ping: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started ping process: %s\n", uuid)
		}

		uuid, err = pm.StartProcess("notepad", []string{}, false)
		if err != nil {
			log.Printf("Error starting notepad: %v", err)
			// Fallback to a Windows command
			uuid, err = pm.StartProcess("cmd", []string{"/c", "timeout", "30"}, false)
			if err == nil {
				processUUIDs = append(processUUIDs, uuid)
				fmt.Printf("Started cmd process: %s\n", uuid)
			}
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started notepad process: %s\n", uuid)
		}
	} else {
		// Unix examples
		uuid, err := pm.StartProcess("ping", []string{"127.0.0.1", "-c", "10"}, true)
		if err != nil {
			log.Printf("Error starting ping: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started ping process: %s\n", uuid)
		}

		uuid, err = pm.StartProcess("sleep", []string{"30"}, false)
		if err != nil {
			log.Printf("Error starting sleep: %v", err)
		} else {
			processUUIDs = append(processUUIDs, uuid)
			fmt.Printf("Started sleep process: %s\n", uuid)
		}
	}

	// Display all running processes
	fmt.Println("\nCurrent processes:")
	for _, process := range pm.ListProcesses() {
		fmt.Printf("  UUID: %s, Name: %s, PID: %d, Status: %s, Uptime: %v\n",
			process.UUID, process.Name, process.PID, process.Status(), process.Uptime())
	}

	// Wait a bit then restart the first process
	if len(processUUIDs) > 0 {
		time.Sleep(3 * time.Second)

		fmt.Printf("\nRestarting process: %s\n", processUUIDs[0])
		newUUID, err := pm.RestartProcess(processUUIDs[0])
		if err != nil {
			log.Printf("Error restarting process: %v", err)
		} else {
			fmt.Printf("Successfully restarted as: %s\n", newUUID)
			processUUIDs[0] = newUUID
		}
	}

	// Display updated process list
	fmt.Println("\nUpdated processes:")
	for _, process := range pm.ListProcesses() {
		fmt.Printf("  UUID: %s, Name: %s, PID: %d, Status: %s, Restart Count: %d\n",
			process.UUID, process.Name, process.PID, process.Status(), process.RestartCount)
	}

	// Interactive management demo
	fmt.Println("\nProcess manager is running. You can:")
	fmt.Println("- Let processes run naturally")
	fmt.Println("- Press Ctrl+C to gracefully shutdown")
	fmt.Println("- The ping process will auto-restart when it completes")

	// Keep the main goroutine alive
	select {}
}
