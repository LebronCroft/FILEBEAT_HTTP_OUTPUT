package main

import (
	"fmt"
	"github.com/fufuok/beats-http-output/config"
	infraLog "github.com/fufuok/beats-http-output/infra"
	_ "github.com/fufuok/beats-http-output/libbeat/outputs/http"
	"github.com/fufuok/beats-http-output/send_bussiness"
	"os"
	"os/exec"
)

var fileBeatConfigFile = "/filebeat.yml"
var agentIDAddress = "/etc/one-agent/machine-id"

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[ERROR] Failed to get working directory: %v\n", err)
		return
	}

	// 拼接 filebeat.yml 的绝对路径
	configFile := wd + fileBeatConfigFile

	// 获取 Agent ID
	infraLog.GlobalLog.Info("Starting to get Agent ID...")
	agentID, err := filebeatConfig.GetAgentID(agentIDAddress)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("[FATAL] GetAgentID failed: %s", err.Error()))
		return
	}
	// 生成配置文件
	jsonData := []byte(`
[
    {
        "module_name": "nginx",
        "mode": 0
    }
]
`)

	config, err := filebeatConfig.GenerateFilebeatConfig(jsonData, agentID)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Error generating filebeat config: %v", err))
		return
	}
	infraLog.GlobalLog.Info("Filebeat config generated successfully.")

	// 写入配置文件
	infraLog.GlobalLog.Info("Writing generated config to filebeat.yml...")
	fmt.Println("config", config)
	err = filebeatConfig.WriteConfigToFile(config, configFile)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Error writing config to file: %v", err))
	} else {
		infraLog.GlobalLog.Info("Filebeat config written successfully.")
	}

	infraLog.GlobalLog.Info(fmt.Sprintf("Successfully retrieved Agent ID: %s", agentID))
	// 启动 Filebeat
	infraLog.GlobalLog.Info("Starting Filebeat with config file: " + configFile)

	startFileBeat(configFile)
	// 循环监听任务
	for {
		infraLog.GlobalLog.Info("Listening for new tasks...")
		pluginsTask, err := send_bussiness.PluginClient.ReceiveTask()
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Error receiving task: %v", err))
			return
		}
		if pluginsTask == nil {
			infraLog.GlobalLog.Info("No tasks received, continuing to listen...")
			continue // 如果没有任务则继续监听
		}
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task:%+x", pluginsTask))
		// 检查任务类型，确保是 ConfigDataType 类型
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task with DataType: %d", pluginsTask.DataType))
		if pluginsTask.DataType != int32(send_bussiness.ConfigDataType) {
			infraLog.GlobalLog.Info(fmt.Sprintf("Received task is not relevant (DataType: %d), skipping...", pluginsTask.DataType))
			continue // 跳过不相关的任务，继续下一次监听
		}

		// 生成配置文件
		infraLog.GlobalLog.Info("Generating filebeat config from received task")
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task:%+x", pluginsTask))
		config, err := filebeatConfig.GenerateFilebeatConfig([]byte(pluginsTask.Data), agentID)
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Error generating filebeat config: %v", err))
			return
		}
		infraLog.GlobalLog.Info("Filebeat config generated successfully.")

		// 写入配置文件
		infraLog.GlobalLog.Info("Writing generated config to filebeat.yml...")
		err = filebeatConfig.WriteConfigToFile(config, fileBeatConfigFile)
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Error writing config to file: %v", err))
		} else {
			infraLog.GlobalLog.Info("Filebeat config written successfully.")
		}

		// 发送任务状态
		status := "succeed"
		msg := ""
		if err != nil {
			status = "failed"
			msg = err.Error()
		}
		infraLog.GlobalLog.Info(fmt.Sprintf("Sending task status (%s)...", status))
		send_bussiness.TaskStatusSendServer(status, pluginsTask.Token, msg)
	}
}

func startFileBeat(configFile string) {

	// 启动新的 filebeat 进程
	currentCmd := exec.Command("./filebeatexc", "-e", "-c", configFile)
	infraLog.GlobalLog.Info("Starting filebeat plugin process...")
	err := currentCmd.Start()
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
		return
	}
	infraLog.GlobalLog.Info("Filebeat process started successfully.")

	// 等待进程结束
	err = currentCmd.Wait()
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat process exited with error: %v", err))
	}
}
