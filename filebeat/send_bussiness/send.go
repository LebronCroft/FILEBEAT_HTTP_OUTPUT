package send_bussiness

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/v7/filebeat/cmd"
	inputs "github.com/elastic/beats/v7/filebeat/input/default-inputs"
	plugins "github.com/elastic/beats/v7/filebeat/send_bussiness/lib/go"
	infraLog "github.com/elastic/beats/v7/infra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func init() {
	PluginClient = plugins.New()
	return
}

var (
	BaseLineDataType = 7000
	PluginClient     *plugins.Client
)

type RetBaselineInfo struct {
	Status        string         `json:"status" bson:"status"`
	LogEntryInfos []LogEntryInfo `json:"check_list" bson:"check_list"`
}

type FilebeatConfig struct {
	Inputs []InputConfig `yaml:"filebeat.inputs"`
	// 其他需要的配置字段
}

type InputConfig struct {
	Type    string   `yaml:"type"`
	Enabled bool     `yaml:"enabled"`
	Paths   []string `yaml:"paths"`
	// 其他需要的字段
}

// 查找最新的日志文件
func findLatestLogFile(dir, pattern string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", fmt.Errorf("error listing files: %v", err)
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no log files found matching pattern %s", pattern)
	}

	// 按文件名排序，确保最新的文件在最后
	sort.Strings(files)
	return files[len(files)-1], nil
}
func SendServer(LogEntryInfo []LogEntryInfo, token string) (err error) {
	var dataInfo = []byte("") // 空的 JSON 对象作为初始值

	infraLog.GlobalLog.Info(fmt.Sprintf("Preparing to send %d log entries to server", len(LogEntryInfo)))
	record := plugins.Record{}
	record.DataType = int32(BaseLineDataType)
	record.Timestamp = time.Now().Unix()
	if len(LogEntryInfo) != 0 {
		dataInfo, err = json.Marshal(LogEntryInfo)
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Failed to marshal log entries: %v", err))
			return err
		}
	}

	infraLog.GlobalLog.Debug(fmt.Sprintf("Marshalled data size: %d bytes", len(dataInfo)))
	infraLog.GlobalLog.Debug(fmt.Sprintf("Marshalled data: %s", string(dataInfo)))

	payload := plugins.Payload{}
	field := make(map[string]string, 0)
	field["data"] = string(dataInfo)
	field["token"] = token

	payload.Fields = field
	record.Data = &payload

	infraLog.GlobalLog.Debug("Calling SendRecord...")

	// 捕获 EOF 错误
	err = PluginClient.SendRecord(&record)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to send record to server: %v", err))
		return err
	}
	infraLog.GlobalLog.Debug("SendRecord completed.")
	infraLog.GlobalLog.Info("Successfully sent data to server")
	return nil
}

func updateFilebeatConfig(logPath string) error {
	// 读取现有的Filebeat配置文件
	configData, err := ioutil.ReadFile("./filebeat.yml")
	if err != nil {
		return fmt.Errorf("读取Filebeat配置文件失败: %v", err)
	}

	// 解析YAML配置到Go结构体
	var config FilebeatConfig
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return fmt.Errorf("解析Filebeat配置文件失败: %v", err)
	}

	// 更新日志采集路径
	if len(config.Inputs) > 0 {
		config.Inputs[0].Paths = []string{logPath}
	} else {
		// 如果没有输入配置，创建一个新的
		newInput := InputConfig{
			Type:    "filestream",
			Enabled: true,
			Paths:   []string{logPath},
		}
		config.Inputs = append(config.Inputs, newInput)
	}

	// 将更新后的配置序列化回YAML
	updatedConfigData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("序列化更新后的配置失败: %v", err)
	}

	// 将更新后的配置写回文件
	err = ioutil.WriteFile("./filebeat.yml", updatedConfigData, 0644)
	if err != nil {
		return fmt.Errorf("写入更新后的配置文件失败: %v", err)
	}

	return nil
}

func restartFilebeat() error {
	// 使用命令行重启Filebeat服务
	command := exec.Command("systemctl", "restart", "filebeat")
	err := command.Run()
	if err != nil {
		return fmt.Errorf("重启Filebeat失败: %v", err)
	}
	return nil
}

// 启动 Filebeat 收集日志数据的函数
func StartFilebeatLogCollector(ctx context.Context) {
	infraLog.GlobalLog.Info("Starting Filebeat log collector...")

	// 监听 context 的取消信号
	go func() {
		<-ctx.Done() // 等待取消信号
		infraLog.GlobalLog.Info("Received stop signal. Stopping Filebeat log collector.")
		// 在这里执行文件收集器的停止操作
		// 比如关闭文件句柄等
	}()

	// 启动 Filebeat 进行日志收集
	if err := cmd.Filebeat(inputs.Init, cmd.FilebeatSettings("")).Execute(); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat failed to start: %v", err))
		os.Exit(1)
	}
	infraLog.GlobalLog.Info("Filebeat log collector started successfully")

	// 其他逻辑保持不变
}

// Simplified data processing logic with continuous loop
func StartDataProcessing(pluginsTask *plugins.Task, ctx context.Context) {

	infraLog.GlobalLog.Info("Starting data processing...")

	// The loop will keep running until we get a stop signal
	for {
		select {
		case <-ctx.Done(): // If context is canceled, stop the loop
			infraLog.GlobalLog.Info("Received stop signal. Stopping data processing.")
			return
		default:
			// Start reading and processing log files
			infraLog.GlobalLog.Info("Processing new batch of log data...")

			// 设定日志文件目录和匹配模式
			dir := "./output"
			baseFileName := "log_output" // 基本文件名，无需包含日期或版本号

			// 删除今天以前的日志文件
			err := deleteOldLogFiles(dir, baseFileName)
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Error deleting old log files: %v", err))
				return
			}

			// 读取并处理日志文件
			logEntries, err := IncrementalRead(dir, baseFileName)
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Error reading log file: %v", err))
				return
			}

			if len(logEntries) > 0 {
				// 发送日志数据到服务器
				err = SendServer(logEntries, pluginsTask.Token)
				if err != nil {
					infraLog.GlobalLog.Error(fmt.Sprintf("Error sending data to server: %v", err))
				} else {
					infraLog.GlobalLog.Info("Successfully sent data to server")
				}
			}

			// 等待一定时间后继续下一次数据读取
			time.Sleep(5 * time.Second) // Adjust this interval as needed
		}
	}
}

// deleteOldLogFiles 删除今天以前的日志文件
func deleteOldLogFiles(dir string, baseFileName string) error {
	// 检查目录是否存在
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// 目录不存在，输出日志并返回
		infraLog.GlobalLog.Info(fmt.Sprintf("Directory %s does not exist, skipping log deletion.", dir))
		return nil
	} else if err != nil {
		// 如果出现其他错误，也记录日志
		infraLog.GlobalLog.Error(fmt.Sprintf("Error checking directory %s: %v", dir, err))
		return err
	}
	// 获取当前日期
	now := time.Now()
	today := now.Format("20060102") // 使用日期格式为 "YYYYMMDD"

	// 遍历文件目录
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只删除 log_output 开头的文件，并且文件名包含日期部分
		if strings.HasPrefix(info.Name(), baseFileName) {
			// 提取日期部分（如 "log_output_20230101" -> "20230101"）
			parts := strings.Split(info.Name(), "_")
			if len(parts) > 1 {
				fileDate := parts[1] // 获取日期部分

				// 如果文件日期早于今天，则删除该文件
				if fileDate < today {
					infraLog.GlobalLog.Info(fmt.Sprintf("Deleting old log file: %s", path))
					err := os.Remove(path)
					if err != nil {
						infraLog.GlobalLog.Error(fmt.Sprintf("Error deleting file %s: %v", path, err))
					}
				}
			}
		}

		return nil
	})

	return err
}
