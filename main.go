package main

import (
	"fmt"
	filebeatConfig "github.com/fufuok/beats-http-output/config"
	infraLog "github.com/fufuok/beats-http-output/infra"
	_ "github.com/fufuok/beats-http-output/libbeat/outputs/http"
	"github.com/fufuok/beats-http-output/send_bussiness"
	"os"
	"os/exec"
	"sync"
)

var fileBeatConfigFile = "/filebeat.yml"
var agentIDAddress = "/etc/one-agent/machine-id"
var mu sync.Mutex // 全局锁

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

	infraLog.GlobalLog.Info("Filebeat config generated successfully.")

	// 写入配置文件
	err = filebeatConfig.UpdateAgentIDInConfigFile(configFile, agentID)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Error writing agentID to file: %v", err))
	} else {
		infraLog.GlobalLog.Info("Filebeat config written successfully.")
	}

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
		//todo 生成配置文件到modules.d文件夹下面的filebeat.yml
		// 生成配置文件
		//infraLog.GlobalLog.Info("Generating filebeat config from received task")
		//infraLog.GlobalLog.Info(fmt.Sprintf("Received task:%+x", pluginsTask))
		//config, err := filebeatConfig.GenerateFilebeatConfig([]byte(pluginsTask.Data), agentID)
		//if err != nil {
		//	infraLog.GlobalLog.Error(fmt.Sprintf("Error generating filebeat config: %v", err))
		//	return
		//}
		//infraLog.GlobalLog.Info("Filebeat config generated successfully.")
		//
		//// 写入配置文件
		//infraLog.GlobalLog.Info("Writing generated config to filebeat.yml...")
		//err = filebeatConfig.WriteConfigToFile(config, fileBeatConfigFile)
		//if err != nil {
		//	infraLog.GlobalLog.Error(fmt.Sprintf("Error writing config to file: %v", err))
		//} else {
		//	infraLog.GlobalLog.Info("Filebeat config written successfully.")
		//}

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

func startFileBeat(configFile string) *exec.Cmd {
	mu.Lock()
	defer mu.Unlock()

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

	//go func() {
	//	err := cmd.Wait()
	//	if err != nil {
	//		infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat process exited with error: %v", err))
	//	}
	//	infraLog.GlobalLog.Info("Filebeat process exited. Attempting to restart...")
	//	startFileBeat(configFile)
	//}()

	infraLog.GlobalLog.Info("Filebeat process started successfully.")
	return cmd
}

//func monitorConfigChanges(configFile string) {
//	watcher, err := fsnotify.NewWatcher()
//	if err != nil {
//		infraLog.GlobalLog.Error(fmt.Sprintf("Error initializing file watcher: %v", err))
//		return
//	}
//	defer watcher.Close()
//
//	// 添加文件到监控列表
//	fmt.Println("Monitoring file:", configFile)
//	err = watcher.Add(configFile)
//	if err != nil {
//		infraLog.GlobalLog.Error(fmt.Sprintf("Error adding file to watcher: %v", err))
//		return
//	}
//
//	// 创建阻塞通道，防止主程序退出
//	done := make(chan bool)
//
//	go func() {
//		for {
//			select {
//			case event := <-watcher.Events:
//				// 检测文件变动
//				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
//					infraLog.GlobalLog.Info(fmt.Sprintf("File %s has changed, restarting filebeat...", event.Name))
//
//					// 停止当前的 filebeat 进程
//					if currentCmd != nil && currentCmd.Process != nil {
//						err := currentCmd.Process.Kill()
//						if err != nil {
//							infraLog.GlobalLog.Error(fmt.Sprintf("Failed to kill current filebeat process: %v", err))
//						} else {
//							infraLog.GlobalLog.Info("Successfully killed current filebeat process.")
//						}
//
//						// 回收资源
//						err = currentCmd.Wait()
//						if err != nil {
//							infraLog.GlobalLog.Error(fmt.Sprintf("Error while waiting for filebeat process to terminate: %v", err))
//						} else {
//							infraLog.GlobalLog.Info("Filebeat process resources cleaned up.")
//						}
//					}
//					time.Sleep(3 * time.Second)
//					// 重启 filebeat 进程
//					if err := killFilebeatProcess(); err != nil {
//						infraLog.GlobalLog.Error(fmt.Sprintf("Error killing existing Filebeat process: %v", err))
//						return
//					}
//
//					cleanupLockFile(dataPath)
//					currentCmd = startFileBeat(configFile)
//				}
//			case err := <-watcher.Errors:
//				infraLog.GlobalLog.Error(fmt.Sprintf("Watcher error: %v", err))
//			}
//		}
//	}()
//
//	// 阻止主程序退出
//	<-done
//}
//
//func killFilebeatProcess() error {
//	cmd := exec.Command("pkill", "-f", "filebeatexc")
//	err := cmd.Run()
//	if err != nil {
//		return fmt.Errorf("failed to kill Filebeat process: %v", err)
//	}
//	return nil
//}
//
//func cleanupLockFile(dataPath string) {
//	lockFile := dataPath + "/filebeat.lock"
//	if _, err := os.Stat(lockFile); err == nil {
//		err = os.Remove(lockFile)
//		if err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("Failed to remove lock file: %v", err))
//		} else {
//			infraLog.GlobalLog.Info("Lock file removed successfully.")
//		}
//	} else {
//		infraLog.GlobalLog.Info("No lock file found to remove.")
//	}
//}
