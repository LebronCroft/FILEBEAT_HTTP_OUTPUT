package main

import (
	"fmt"
	"github.com/fufuok/beats-http-output/filebeat/send_bussiness"
	infraLog "github.com/fufuok/beats-http-output/infra"
	"os/exec"
	"runtime"
	"runtime/debug"
	_ "time/tzdata" // for timezone handling
)

// 初始化 Go 协程数量
func init() {
	runtime.GOMAXPROCS(4)
}

func main() {
	var currentCmd *exec.Cmd
	var configFile = "./filebeat.yml"

	defer func() {
		if err := recover(); err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("recover panic: %v, %s", err, debug.Stack()))
		}
	}()

	go func() {
		startFileBeat(configFile)
	}()

	for {
		// 监听接收任务
		pluginsTask, err := send_bussiness.PluginClient.ReceiveTask()
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Error receiving task: %v", err))
			return
		}
		if pluginsTask == nil {
			continue // 如果没有任务则继续监听
		}

		// 只处理 DataType 为 ConfigDataType 的任务
		if pluginsTask.DataType != int32(send_bussiness.ConfigDataType) {
			infraLog.GlobalLog.Info(fmt.Sprintf("Received task is not relevant: %v", pluginsTask))
			continue // 跳过不相关的任务，继续下一次监听
		}
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task: %v", pluginsTask))
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task Data: %v", pluginsTask.Data))
		// 解析任务中的 data 字段，提取日志文件路径
		logPath, err := send_bussiness.ParseTaskData(pluginsTask.Data)
		infraLog.GlobalLog.Info(fmt.Sprintf("logPath: %v", logPath))
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Failed to parse task data: %v", err))
			continue
		}
		// 接收到符合条件的任务时，打印日志并执行相关处理
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task: %v", pluginsTask))

		// 1. 检查并停止现有的 filebeat 进程（如果有的话）
		if currentCmd != nil && currentCmd.Process != nil {
			infraLog.GlobalLog.Info("Stopping previous filebeat process...")
			err = currentCmd.Process.Kill()
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to stop previous filebeat process: %v", err))
			} else {
				infraLog.GlobalLog.Info("Successfully stopped previous filebeat process.")
			}
			// 2. 更新 filebeat 配置文件
			err = send_bussiness.UpdateFilebeatConfig(configFile, logPath)
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to update filebeat config: %v", err))
			}
			startFileBeat(configFile)
		}

		msg := ""
		status := "succeed"
		if err != nil {
			status = "failed"
			msg = err.Error()
		}
		send_bussiness.TaskStatusSendServer(status, pluginsTask.Token, msg)
	}
}

func startFileBeat(configFile string) {
	currentCmd := exec.Command("./filebeatexc", "-e", "-c", configFile)
	infraLog.GlobalLog.Info("Starting filebeat plugin process...")

	err := currentCmd.Start()
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
		return
	}
	infraLog.GlobalLog.Info("Filebeat process started successfully.")
}
