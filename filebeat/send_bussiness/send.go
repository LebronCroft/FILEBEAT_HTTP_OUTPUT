package send_bussiness

import (
	"encoding/json"
	"fmt"
	plugins "github.com/fufuok/beats-http-output/filebeat/send_bussiness/lib/go"
	infraLog "github.com/fufuok/beats-http-output/infra"

	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	PluginClient = plugins.New()
	return
}

var (
	ConfigDataType = 7000
	ReportDataType = 7001
	PluginClient   *plugins.Client
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

func SendServer(LogEntryInfo []LogEntryInfo, token string) (err error) {

	var dataInfo = []byte("") // 空的 JSON 对象作为初始值

	infraLog.GlobalLog.Info(fmt.Sprintf("Preparing to send %d log entries to server", len(LogEntryInfo)))
	record := plugins.Record{}
	record.DataType = int32(ReportDataType)
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

// Simplified data processing logic with continuous loop
func StartDataProcessing() {

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
			err = SendServer(logEntries, "0")
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
	infraLog.GlobalLog.Info(fmt.Sprintf("Parsing task data %s", data))
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

func UpdateFilebeatConfig(configFile, agentID string) error {
	// 打开并解析现有的 filebeat.yml 配置文件
	infraLog.GlobalLog.Info("Opening filebeat.yml configuration file...")
	file, err := os.OpenFile(configFile, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open filebeat.yml: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			// 如果有错误，可以选择记录或处理
		}
	}(file)

	var config map[string]interface{}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to decode filebeat.yml: %v", err)
	}

	// 打印整个配置，以便调试
	infraLog.GlobalLog.Info(fmt.Sprintf("Decoded config: %+v", config))

	// 查找并检查 filebeat 字段是否存在并正确解析
	filebeatField, ok := config["filebeat"]
	if !ok {
		infraLog.GlobalLog.Error("filebeat field is missing in the config.")
		return fmt.Errorf("filebeat field is missing in the config")
	}

	// 打印 filebeat 字段，看看它的类型和内容
	infraLog.GlobalLog.Info(fmt.Sprintf("filebeat field type: %T, value: %+v", filebeatField, filebeatField))

	// 检查 filebeat 字段是否是一个 map 类型
	filebeatConfig, ok := filebeatField.(map[interface{}]interface{})
	if !ok {
		infraLog.GlobalLog.Error("filebeat field is not a map[interface{}]interface{}")
		return fmt.Errorf("filebeat field is not a map[interface{}]interface{}")
	}

	// 查找并更新 fields 配置
	fieldsConfig, ok := filebeatConfig["inputs"].([]interface{})
	if !ok {
		infraLog.GlobalLog.Error("filebeat.inputs is missing or not a slice.")
		return fmt.Errorf("filebeat.inputs is missing or not a slice.")
	}

	// 假设您只需要修改第一个 inputs 条目
	if len(fieldsConfig) > 0 {
		// 更新 fields 中的 agentID 字段
		input := fieldsConfig[0].(map[interface{}]interface{})
		if input["fields"] == nil {
			// 如果没有找到 fields 配置，则创建它
			input["fields"] = map[interface{}]interface{}{
				"agentID": agentID,
			}
		} else {
			// 更新现有的 fields 配置
			fields := input["fields"].(map[interface{}]interface{})
			fields["agentID"] = agentID
		}
	} else {
		// 如果没有找到任何 inputs 配置，则创建一个新的配置
		filebeatConfig["inputs"] = []interface{}{
			map[interface{}]interface{}{
				"type":    "filestream",
				"enabled": true,
				"fields": map[interface{}]interface{}{
					"agentID": agentID,
				},
				"paths": []interface{}{" /var/log/nginx/access.log.1"},
			},
		}
	}

	// 打印更新后的配置
	infraLog.GlobalLog.Info(fmt.Sprintf("Updated config: %+v", config))

	// 写回更新后的配置到文件
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	} // 重置文件指针
	err = file.Truncate(0)
	if err != nil {
		return err
	} // 清空文件内容
	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(config)
	if err != nil {
		return fmt.Errorf("failed to write updated config to filebeat.yml: %v", err)
	}

	infraLog.GlobalLog.Info(fmt.Sprintf("filebeat.yml updated with agentID: %v", agentID))
	return nil
}


// TaskStatusSendServer send task result to server
func TaskStatusSendServer(status string, token string, msg string) {
	record := plugins.Record{}
	record.DataType = int32(ConfigDataType)
	record.Timestamp = time.Now().Unix()

	payload := plugins.Payload{}
	field := make(map[string]string, 0)
	field["status"] = status
	if token != "" {
		field["token"] = token
	}
	field["msg"] = msg
	payload.Fields = field
	record.Data = &payload

	_ = PluginClient.SendRecord(&record)
}
