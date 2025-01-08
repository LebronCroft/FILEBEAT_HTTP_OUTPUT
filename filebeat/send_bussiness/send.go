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
	"os"
	"path/filepath"
	"regexp"
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
	// 然后进行 JSON 编码
	dataInfo, err = json.Marshal(LogEntryInfo)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to marshal log entries: %v", err))
		return err
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

// CleanText 去除所有控制字符和不可见字符
func CleanText(input string) string {
	// 正则表达式匹配所有非打印字符
	re := regexp.MustCompile(`[\x00-\x1F\x7F\uFFFD]`) // 匹配控制字符和 \ufffd 替代字符
	cleaned := re.ReplaceAllString(input, "")         // 替换为 ""
	return cleaned
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
	if _, err := cmd.Filebeat(inputs.Init, cmd.FilebeatSettings("")).ExecuteContextC(ctx); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat failed to start: %v", err))
		os.Exit(1)
	}
	infraLog.GlobalLog.Info("Filebeat log collector started successfully")

	time.AfterFunc(time.Second*10, func() {
		ctx.Done()
	})
	// 其他逻辑保持不变
}

// Simplified data processing logic with continuous loop
func StartDataProcessing(pluginsTask *plugins.Task) {

	infraLog.GlobalLog.Info("Starting data processing...")
	for {
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

// 解析任务中的 data 字段，提取 log_address
func ParseTaskData(data interface{}) (string, error) {
	// 如果 data 是字符串类型，首先解析该字符串为 map
	switch v := data.(type) {
	case string:
		// 解析 JSON 字符串为 map
		var dataMap map[string]interface{}
		err := json.Unmarshal([]byte(v), &dataMap)
		if err != nil {
			return "", fmt.Errorf("failed to parse JSON string: %v", err)
		}

		// 提取 log_address
		logAddress, ok := dataMap["log_address"].(string)
		if !ok {
			return "", fmt.Errorf("log_address field is missing or not a string")
		}
		return logAddress, nil

	case map[string]interface{}:
		// 如果 data 本身是一个 map，直接提取 log_address
		logAddress, ok := v["log_address"].(string)
		if !ok {
			return "", fmt.Errorf("log_address field is missing or not a string")
		}
		return logAddress, nil

	default:
		return "", fmt.Errorf("failed to parse task data: expected string or map, got %T", v)
	}
}

func UpdateFilebeatConfig(configFile, logPath string) error {
	// 打开并解析现有的 filebeat.yml 配置文件
	file, err := os.OpenFile(configFile, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open filebeat.yml: %v", err)
	}
	defer file.Close()

	var config map[string]interface{}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to decode filebeat.yml: %v", err)
	}

	// 查找并更新 filebeat.inputs 配置
	filebeatInputs, ok := config["filebeat"].(map[string]interface{})["inputs"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to find filebeat inputs in the config")
	}

	// 假设你只需要修改第一个 inputs 条目
	if len(filebeatInputs) > 0 {
		input := filebeatInputs[0].(map[string]interface{})
		input["paths"] = []string{logPath} // 更新为新的日志路径
	} else {
		// 如果没有找到任何 inputs 配置，则创建一个新的配置
		config["filebeat"] = map[string]interface{}{
			"inputs": []interface{}{
				map[string]interface{}{
					"type":          "filestream",
					"enabled":       true,
					"paths":         []string{logPath},
					"max_line_size": 10485760, // 10MB
				},
			},
		}
	}

	// 写回更新后的配置到文件
	file.Seek(0, 0)  // 重置文件指针
	file.Truncate(0) // 清空文件内容
	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(config)
	if err != nil {
		return fmt.Errorf("failed to write updated config to filebeat.yml: %v", err)
	}

	infraLog.GlobalLog.Info(fmt.Sprintf("filebeat.yml updated with log path: %v", logPath))
	return nil
}
