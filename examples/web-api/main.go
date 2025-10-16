package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/yourusername/process-manager/manager"
)

var pm *manager.ProcessManager

func main() {
	pm = manager.NewProcessManager()
	defer pm.Shutdown()

	// Setup HTTP routes
	http.HandleFunc("/processes", listProcesses)
	http.HandleFunc("/process/start", startProcess)
	http.HandleFunc("/process/stop", stopProcess)
	http.HandleFunc("/process/restart", restartProcess)

	fmt.Println("Process Manager API server running on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /processes - List all processes")
	fmt.Println("  POST /process/start - Start a new process")
	fmt.Println("  POST /process/stop - Stop a process")
	fmt.Println("  POST /process/restart - Restart a process")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func listProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	processes := pm.ListProcesses()
	json.NewEncoder(w).Encode(processes)
}

func startProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Name    string   `json:"name"`
		Args    []string `json:"args"`
		Restart bool     `json:"restart"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	uuid, err := pm.StartProcess(request.Name, request.Args, request.Restart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"uuid": uuid}
	json.NewEncoder(w).Encode(response)
}

func stopProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		UUID string `json:"uuid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := pm.StopProcess(request.UUID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func restartProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		UUID string `json:"uuid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newUUID, err := pm.RestartProcess(request.UUID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"new_uuid": newUUID}
	json.NewEncoder(w).Encode(response)
}
