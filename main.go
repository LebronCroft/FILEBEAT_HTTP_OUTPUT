package main

import (
	"fmt"
	"github.com/elastic/beats/v7/filebeat/send_bussiness"
	infraLog "github.com/elastic/beats/v7/infra"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sync"
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

	var wg sync.WaitGroup

	// 启动主循环，监听插件任务
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

		// 只处理 DataType 为 BaseLineDataType 的任务
		if pluginsTask.DataType != int32(send_bussiness.BaseLineDataType) {
			infraLog.GlobalLog.Info(fmt.Sprintf("Received task is not relevant: %v", pluginsTask))
			continue // 跳过不相关的任务，继续下一次监听
		}

		//// 解析任务中的 data 字段，提取日志文件路径
		//logPath, err := send_bussiness.ParseTaskData(pluginsTask.Data)
		//if err != nil {
		//	infraLog.GlobalLog.Error(fmt.Sprintf("Failed to parse task data: %v", err))
		//	continue
		//}
		// 接收到符合条件的任务时，打印日志并执行相关处理
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task: %v", pluginsTask))
		// 1. 检查并停止现有的 filebeat 进程（如果有的话）
		if currentCmd != nil && currentCmd.Process != nil {
			infraLog.GlobalLog.Info("Stopping previous filebeat process...")
			err := currentCmd.Process.Kill()
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to stop previous filebeat process: %v", err))
			} else {
				infraLog.GlobalLog.Info("Successfully stopped previous filebeat process.")
			}
		}
		// 1. 检查并停止现有的 filebeat 进程（如果有的话）
		if currentCmd != nil && currentCmd.Process != nil {
			infraLog.GlobalLog.Info("Stopping previous filebeat process...")
			err := currentCmd.Process.Kill()
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to stop previous filebeat process: %v", err))
			} else {
				infraLog.GlobalLog.Info("Successfully stopped previous filebeat process.")
			}
		}
		// 重新启动处理程序
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 1. 检查并停止现有的 filebeat 进程（如果有的话）
			if currentCmd != nil && currentCmd.Process != nil {
				infraLog.GlobalLog.Info("Stopping previous filebeat process...")
				err := currentCmd.Process.Kill()
				if err != nil {
					infraLog.GlobalLog.Error(fmt.Sprintf("Failed to stop previous filebeat process: %v", err))
				} else {
					infraLog.GlobalLog.Info("Successfully stopped previous filebeat process.")
				}
			}

			//// 2. 更新 filebeat 配置文件
			//err = send_bussiness.UpdateFilebeatConfig(configFile, logPath)
			//if err != nil {
			//	infraLog.GlobalLog.Error(fmt.Sprintf("Failed to update filebeat config: %v", err))
			//	return
			//}
			// 3. 启动新的 filebeat 进程
			currentCmd = exec.Command("./filebeatexc", "-e", "-c", configFile)
			infraLog.GlobalLog.Info("Starting filebeat plugin process...")

			err = currentCmd.Start()
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
				return
			}
			infraLog.GlobalLog.Info("Filebeat process started successfully.")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			// 启动数据处理任务
			send_bussiness.StartDataProcessing(pluginsTask)
		}()

		// 等待两个子任务完成
		wg.Wait()
	}
}

//func main() {
//	var currentCmd *exec.Cmd
//	var configFile = "./filebeat.yml"
//
//	// 启动主循环，监听插件任务
//	for {
//		// 监听接收任务
//		pluginsTask, err := send_bussiness.PluginClient.ReceiveTask()
//		if err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("Error receiving task: %v", err))
//			continue
//		}
//		if pluginsTask == nil {
//			continue // 如果没有任务则继续监听
//		}
//
//		// 只处理 DataType 为 BaseLineDataType 的任务
//		if pluginsTask.DataType != int32(send_bussiness.BaseLineDataType) {
//			infraLog.GlobalLog.Info(fmt.Sprintf("Received task is not relevant: %v", pluginsTask))
//			continue // 跳过不相关的任务，继续下一次监听
//		}
//
//		// 接收到符合条件的任务时，打印日志并执行相关处理
//		infraLog.GlobalLog.Info(fmt.Sprintf("Received task: %v", pluginsTask))
//		infraLog.GlobalLog.Info("Entering main loop...")
//
//		// 1. 检查当前是否有正在运行的 filebeat 进程，如果有就停止它
//		if currentCmd != nil && currentCmd.Process != nil {
//			infraLog.GlobalLog.Info("Checking if a filebeat process is already running...")
//
//			// 如果当前 filebeat 进程存在，则先停止它
//			infraLog.GlobalLog.Info("Stopping previous filebeat process...")
//			err := currentCmd.Process.Kill()
//			if err != nil {
//				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to stop previous filebeat process: %v", err))
//			} else {
//				infraLog.GlobalLog.Info("Successfully stopped previous filebeat process.")
//			}
//		}
//
//		// 2. 启动新的 filebeat 进程
//		infraLog.GlobalLog.Info("Preparing to start a new filebeat process...")
//		currentCmd = exec.Command("./filebeatexc", "-e", "-c", configFile)
//		infraLog.GlobalLog.Info(fmt.Sprintf("Filebeat command prepared: %v", currentCmd))
//
//		// 创建管道来捕获 filebeat 的输出
//		rx_r, rx_w, err := os.Pipe()
//		if err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("Failed to create pipe: %v", err))
//			return
//		}
//		infraLog.GlobalLog.Info("Pipe created successfully for capturing output.")
//
//		// 3. 设置 filebeat 进程的标准输出和标准错误输出为管道
//		currentCmd.Stdout = rx_w
//		currentCmd.Stderr = rx_w
//		infraLog.GlobalLog.Info("Standard output and error are redirected to pipe.")
//
//		// 4. 启动新的 filebeat 进程
//		infraLog.GlobalLog.Info("Starting filebeat plugin process...")
//		err = currentCmd.Start()
//		if err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
//			return
//		}
//		infraLog.GlobalLog.Info("Filebeat process started successfully.")
//
//		// 5. 通过管道读取 filebeat 输出并发送数据
//		// 在 go routine 中处理管道读取
//		go func() {
//			infraLog.GlobalLog.Info("Started reading from filebeat output pipe...")
//			buf := make([]byte, 4096)  // 4KB 缓冲区，可以根据需要调整
//
//			var allData []byte
//
//			for {
//				n, err := rx_r.Read(buf)
//				if err != nil {
//					// 如果遇到 EOF 错误，说明读取完成，正常退出
//					if err.Error() == "EOF" {
//						infraLog.GlobalLog.Info("Reached EOF on pipe, no more data.")
//						break
//					} else {
//						infraLog.GlobalLog.Error(fmt.Sprintf("Error reading from filebeat output pipe: %v", err))
//						break
//					}
//				}
//
//				// 追加读取的数据到 allData
//				allData = append(allData, buf[:n]...)
//
//				// 检查是否有足够的数据以进行 JSON 解码
//				if len(allData) > 0 {
//					// 直接在这里处理解析逻辑
//					rawLogData := string(allData)
//					infraLog.GlobalLog.Info(fmt.Sprintf("Raw log data: %s", rawLogData))
//
//					var entry map[string]interface{}
//					err = json.Unmarshal(allData, &entry)
//					if err != nil {
//						infraLog.GlobalLog.Error(fmt.Sprintf("Failed to unmarshal log data: %v", err))
//						continue
//					}
//
//					// 处理日志条目
//					entryInfos := []map[string]interface{}{entry}
//					logEntry, err := send_bussiness.CreateLogEntryInfo(entryInfos)
//					if err != nil {
//						infraLog.GlobalLog.Error(fmt.Sprintf("Failed to create LogEntryInfo: %v", err))
//						continue
//					}
//
//					infraLog.GlobalLog.Info("Successfully unmarshalled log entry.")
//
//					// 调用 SendServer 函数发送日志
//					err = send_bussiness.SendServer(logEntry, pluginsTask.Token)
//					if err != nil {
//						infraLog.GlobalLog.Error(fmt.Sprintf("Error sending log data to server: %v", err))
//					} else {
//						infraLog.GlobalLog.Info("Successfully sent log data to server")
//					}
//
//					// 清空数据缓存，准备读取下一个日志
//					allData = nil
//				}
//			}
//
//			// 读取完成后关闭管道（确保数据都读取完了再关闭）
//			rx_w.Close()
//			infraLog.GlobalLog.Info("Pipe closed.")
//		}()
//
//		// 等待 filebeat 进程完成
//		infraLog.GlobalLog.Info("Waiting for filebeat process to finish...")
//		err = currentCmd.Wait()
//		if err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("filebeat process finished with error: %v", err))
//		} else {
//			infraLog.GlobalLog.Info("filebeat process completed successfully.")
//		}
//
//		// 延迟或者其它条件继续下一次循环
//		infraLog.GlobalLog.Info("Main loop iteration complete.")
//	}
//}
