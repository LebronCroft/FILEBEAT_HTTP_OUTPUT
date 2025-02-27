package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Stats struct {
	CPUValues    []float64
	MemoryValues []float64
	TotalLines   int       // 记录日志总行数
	StartTime    time.Time // 日志起始时间
	EndTime      time.Time // 日志结束时间
}

func main() {
	logFile := "performance.log"
	stats, err := parseLogFile(logFile)
	if err != nil {
		fmt.Printf("Error parsing log file: %v\n", err)
		return
	}

	generateReport(stats)
}

func parseLogFile(filePath string) (Stats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Stats{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var stats Stats
	var firstTimestampFound bool

	// 正则表达式匹配日志内容
	timeRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) - `)
	cpuRegex := regexp.MustCompile(`CPU Usage: ([\d\.]+)%`)
	memRegex := regexp.MustCompile(`Memory Usage: ([\d\.]+)%`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		stats.TotalLines++ // 统计行数

		// 提取时间
		if timeMatch := timeRegex.FindStringSubmatch(line); timeMatch != nil {
			timestamp, err := time.Parse("2006-01-02 15:04:05", timeMatch[1])
			if err == nil {
				if !firstTimestampFound {
					stats.StartTime = timestamp // 记录起始时间
					firstTimestampFound = true
				}
				stats.EndTime = timestamp // 持续更新为最新的时间
			}
		}

		// 提取 CPU 使用率
		if cpuMatch := cpuRegex.FindStringSubmatch(line); cpuMatch != nil {
			cpuValue, _ := strconv.ParseFloat(cpuMatch[1], 64)
			stats.CPUValues = append(stats.CPUValues, cpuValue)
		}

		// 提取内存使用率
		if memMatch := memRegex.FindStringSubmatch(line); memMatch != nil {
			memValue, _ := strconv.ParseFloat(memMatch[1], 64)
			stats.MemoryValues = append(stats.MemoryValues, memValue)
		}
	}

	if err := scanner.Err(); err != nil {
		return Stats{}, fmt.Errorf("error reading file: %w", err)
	}

	return stats, nil
}

func generateReport(stats Stats) {
	cpuMax, cpuMin, cpuAvg := calculateStats(stats.CPUValues)
	memMax, memMin, memAvg := calculateStats(stats.MemoryValues)

	// 计算总时长
	totalDuration := stats.EndTime.Sub(stats.StartTime)

	fmt.Println("=== 性能测试报告 ===")
	fmt.Printf("日志总行数: %d\n", stats.TotalLines) // 新增输出日志总行数
	fmt.Printf("日志时间范围: %s ~ %s\n", stats.StartTime.Format("2006-01-02 15:04:05"), stats.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("总时长: %s\n", totalDuration)
	fmt.Printf("CPU 使用率：最大值 %.2f%%，最小值 %.2f%%，平均值 %.2f%%\n", cpuMax, cpuMin, cpuAvg)
	fmt.Printf("内存使用率：最大值 %.2f%%，最小值 %.2f%%，平均值 %.2f%%\n", memMax, memMin, memAvg)
	fmt.Println("===================")
}

func calculateStats(values []float64) (max, min, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	min = values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
		avg += v
	}
	avg /= float64(len(values))
	return max, min, avg
}
