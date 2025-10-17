package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dreamsxin/process-manager/system"
	"github.com/dreamsxin/process-manager/types"
)

var systemMonitor *system.SystemMonitor

func main() {
	// 创建系统监控器
	systemMonitor = system.NewSystemMonitor("./monitor_data")

	// 启动监控
	if err := systemMonitor.Start(); err != nil {
		log.Fatalf("Failed to start system monitor: %v", err)
	}
	defer systemMonitor.Stop()

	// 设置HTTP路由
	http.HandleFunc("/", serveStatic)
	http.HandleFunc("/api/stats/current", handleCurrentStats)
	http.HandleFunc("/api/stats/history", handleHistory)
	http.HandleFunc("/api/stats/chart", handleChartData)
	http.HandleFunc("/api/alerts", handleAlerts)
	http.HandleFunc("/api/config", handleConfig)

	fmt.Println("System Monitor Server running on :8080")
	fmt.Println("Open http://localhost:8080 in your browser")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// serveStatic 提供静态文件
func serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "examples/system_monitor/index.html")
		return
	}

	// 可以根据需要添加其他静态文件
	http.NotFound(w, r)
}

// handleCurrentStats 返回当前系统统计
func handleCurrentStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := systemMonitor.GetCurrentStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleHistory 返回历史数据
func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	countStr := r.URL.Query().Get("count")
	count := 100 // 默认100条

	if countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Invalid count parameter", http.StatusBadRequest)
			return
		}
	}

	history := systemMonitor.GetHistory(count)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleChartData 返回图表数据
func handleChartData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	countStr := r.URL.Query().Get("count")
	metric := r.URL.Query().Get("metric")

	if metric == "" {
		metric = "all" // 默认显示所有指标
	}

	count := 50 // 默认50条
	if countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Invalid count parameter", http.StatusBadRequest)
			return
		}
	}

	chartData, err := systemMonitor.GetChartData(count, metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartData)
}

// handleAlerts 返回告警信息
func handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alerts := systemMonitor.GetAlerts()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleConfig 处理配置请求
func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config := systemMonitor.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)

	case http.MethodPost:
		var config types.MonitorConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := systemMonitor.UpdateConfig(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
