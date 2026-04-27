//go:build performance
// +build performance

package main

import (
	"fmt"
	"github.com/fufuok/beats-http-output/enum"
	infraLog "github.com/fufuok/beats-http-output/infra"
	"github.com/shirou/gopsutil/process"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var filepath = "/var/log/nginx/access_test.log"

// GenerateLogs 生成指定数量的 nginx 访问日志到指定文件中
func GenerateLogs(filePath string, totalLogs, batchSize int) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	for i := 0; i < totalLogs; i++ {
		logLine := fmt.Sprintf("127.0.0.1 - - [%s +0800] \"GET /static/js/app.js HTTP/1.1\" 200 512 \"http://example.com\" \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/132.0.0.0\"\n", time.Now().Format("02/Jan/2006:15:04:05"))
		if _, err := file.WriteString(logLine); err != nil {
			return fmt.Errorf("failed to write log: %v", err)
		}

		// 每 batchSize 条日志打印一次状态，模拟批量生成
		if i%batchSize == 0 {
			fmt.Printf("Generated %d/%d logs...\n", i, totalLogs)
			time.Sleep(100 * time.Millisecond) // 模拟生成延迟
		}
	}
	fmt.Println("Log generation completed.")
	return nil
}

func StartPerformanceMonitoring() {
	StartFileBeatForTest()
	time.Sleep(5 * time.Second)
	pid, err := getFilebeatPID() // 替换成你的进程名
	if err != nil {
		fmt.Printf("Failed to find process: %v\n", err)
		return
	}
	fmt.Println("pid", pid)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	logFile, _ := os.Create("performance.log")
	defer logFile.Close()
	for range ticker.C {
		cpuUsage, memUsage, memMB, err := getProcessStats(pid)
		if err != nil {
			fmt.Printf("Failed to get process stats: %v\n", err)
			continue
		}

		logLine := fmt.Sprintf("%s - CPU Usage: %.2f%%, Memory Usage: %.2f%% (%.2f MB)\n",
			time.Now().Format("2006-01-02 15:04:05"), cpuUsage, memUsage, memMB)
		fmt.Print(logLine)
		logFile.WriteString(logLine)
	}

}

func getFilebeatPID() (int32, error) {
	output, err := exec.Command("sh", "-c", "ps auf | grep '[.]\\/filebeatexc -e -c .*filebeat.yml'").Output()
	if err != nil || len(output) == 0 {
		return 0, fmt.Errorf("process not found")
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 1 {
			pid, err := strconv.Atoi(fields[1]) // 第二个字段是 PID
			if err == nil {
				return int32(pid), nil
			}
		}
	}
	return 0, fmt.Errorf("failed to parse PID")
}

// getProcessStats 获取进程的 CPU 和内存使用率，以及具体的内存占用（MB）
func getProcessStats(pid int32) (float64, float32, float64, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return 0, 0, 0, err
	}

	cpuPercent, _ := p.CPUPercent()
	memPercent, _ := p.MemoryPercent()

	// 获取进程的内存信息
	memInfo, err := p.MemoryInfo()
	if err != nil {
		return 0, 0, 0, err
	}
	memMB := float64(memInfo.RSS) / 1024 / 1024 // 转换为 MB

	return cpuPercent, memPercent, memMB, nil
}

// StartFileBeatForTest 启动 Filebeat 进行测试
func StartFileBeatForTest() *exec.Cmd {
	wd, _ := os.Getwd()

	// 拼接 filebeat.yml 的绝对路径
	configFile := wd + enum.FileBeatConfigFile
	enum.Mu.Lock()
	defer enum.Mu.Unlock()

	infraLog.GlobalLog.Info("Starting Filebeat with config file: " + configFile)

	if _, err := os.Stat("./filebeatexc"); os.IsNotExist(err) {
		infraLog.GlobalLog.Error("filebeatexc not found in the current directory")
		return nil
	}

	cmd := exec.Command("./filebeatexc", "-e", "-c", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
		return nil
	}

	infraLog.GlobalLog.Info("Filebeat process started successfully.")
	return cmd
}

func main() {

	err := GenerateLogs(filepath, 10000, 1000)
	if err != nil {
		fmt.Println("Error generating log:", err)
	}
	infraLog.GlobalLog.Println("Starting all tasks...")
	// 开始性能监控（异步执行）
	go StartPerformanceMonitoring()

	// 阻塞主线程，防止程序退出
	select {}

}
