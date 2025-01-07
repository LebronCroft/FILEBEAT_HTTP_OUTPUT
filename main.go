package main

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/filebeat/send_bussiness"
	infraLog "github.com/elastic/beats/v7/infra"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
	_ "time/tzdata" // for timezone handling
)

// 初始化 Go 协程数量
func init() {
	runtime.GOMAXPROCS(4)
}

func main() {
	// 创建一个 channel 用于控制任务的停止
	var stopChannel context.CancelFunc

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

		// 接收到符合条件的任务时，打印日志并执行相关处理
		infraLog.GlobalLog.Info(fmt.Sprintf("Received task: %v", pluginsTask))

		// 如果已有 stopChannel，发送取消信号，停止当前任务
		if stopChannel != nil {
			stopChannel() // 取消当前任务
			infraLog.GlobalLog.Info("Sent stop signal to current tasks.")
			// 给一些时间让正在运行的任务完成停止（例如 1 秒）
			time.Sleep(1 * time.Second)
		}

		// 创建新的 context 和 cancel function
		ctx, cancel := context.WithCancel(context.Background())
		stopChannel = cancel // 更新 stopChannel 为新的 cancel function

		// 重新启动处理程序
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 启动日志收集程序
			send_bussiness.StartFilebeatLogCollector(ctx)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			// 启动数据处理任务
			send_bussiness.StartDataProcessing(pluginsTask, ctx)
		}()

		// 等待两个子任务完成
		wg.Wait()
	}
}

//func main() {
//	// 创建一个 channel 用于控制任务的停止
//	stopChannel := make(chan bool)
//
//	defer func() {
//		if err := recover(); err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("recover panic: %v, %s", err, debug.Stack()))
//		}
//	}()
//
//	var wg sync.WaitGroup
//
//	// 启动主循环，监听插件任务
//	for {
//		// 监听接收任务
//		pluginsTask, err := send_bussiness.PluginClient.ReceiveTask()
//		if err != nil {
//			infraLog.GlobalLog.Error(fmt.Sprintf("Error receiving task: %v", err))
//			return
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
//
//		// 重新启动处理程序
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			// 启动日志收集程序
//			send_bussiness.StartFilebeatLogCollector(&wg, stopChannel)
//		}()
//
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			// 启动数据处理任务
//			send_bussiness.StartDataProcessing(&wg, pluginsTask, stopChannel)
//		}()
//
//		// 等待两个子任务完成
//		wg.Wait()
//
//	}
//}
