package main

import (
	"fmt"
	"github.com/fufuok/beats-http-output/config"
	infraLog "github.com/fufuok/beats-http-output/infra"
	"github.com/fufuok/beats-http-output/send_bussiness"
	"os/exec"

	_ "github.com/fufuok/beats-http-output/libbeat/outputs/http"
)

var fileBeatConfigFile = "./filebeat.yml"
var agentIDAddress = "/etc/one-agent/machine-id"

func main() {

	agentID, err := config.GetAgentID(agentIDAddress)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("[FATAL] GetAgentID failed: %s", err.Error()))
	}
	// 启动 Filebeat
	startFileBeat(fileBeatConfigFile)

	// 循环监听接收任务
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

		// 解析任务中的字段并更新配置
		parsedData, err := config.ParseTaskData(pluginsTask.Data)
		if err != nil {
			fmt.Println("Error parsing task data:", err)
			return
		}
		var logType string
		for k, _ := range parsedData {
			logType = k
			break
		}
		filebeatConfig := config.NewFilebeatConfig(agentID, logType)
		// 将配置写入文件
		writeErr := filebeatConfig.WriteToFile(fileBeatConfigFile)
		if writeErr != nil {
			fmt.Printf("Error writing config to file: %v\n", writeErr)
		} else {
			fmt.Println("Filebeat config written successfully!")
		}
		// 发送任务状态
		status := "succeed"
		msg := ""
		if err != nil {
			status = "failed"
			msg = err.Error()
		}
		send_bussiness.TaskStatusSendServer(status, pluginsTask.Token, msg)
	}
}

func startFileBeat(configFile string) {
	currentCmd := exec.Command("./filebeatexc", "-e", "-c", configFile)
	fmt.Println(currentCmd.String())
	infraLog.GlobalLog.Info("Starting filebeat plugin process...")
	fmt.Println("Starting filebeat plugin process...")
	err := currentCmd.Start()
	fmt.Println("Starting filebeat success")
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
		return
	}
	infraLog.GlobalLog.Info("Filebeat process started successfully.")
	// 使用Wait方法阻塞，确保filebeatexc进程持续运行，直到它自己结束或者出现错误
	go func() {
		err := currentCmd.Wait()
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat process exited with error: %v", err))
		}
	}()
}
