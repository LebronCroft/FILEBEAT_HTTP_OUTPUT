package filebeatConfig

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type FilebeatConfig struct {
	Filebeat struct {
		Modules []ModuleConfig `yaml:"modules,omitempty"`
		Inputs  []InputConfig  `yaml:"inputs,omitempty"` // 新增字段，用于支持输入流配置
	} `yaml:"filebeat"`
	Fields FieldsConfig `yaml:"fields"`
	Output OutputConfig `yaml:"output"`
}

type InputConfig struct {
	Type    string   `yaml:"type"`    // 日志类型，这里是 "log"
	Enabled bool     `yaml:"enabled"` // 是否启用
	Paths   []string `yaml:"paths"`   // 日志路径
}

type FieldsConfig struct {
	AgentID string `yaml:"agentID"`
}

type ModuleConfig struct {
	Module string `yaml:"module"`
	Access *struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"access,omitempty"`
	Error *struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"error,omitempty"`
	Logs *struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"logs,omitempty"`
	Slowlog *struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"slowlog,omitempty"`
	Audit *struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"audit,omitempty"`
}

type OutputConfig struct {
	Http struct {
		Hosts []string `yaml:"hosts"`
		Path  string   `yaml:"path"`
	} `yaml:"http"`
}

// 默认路径映射表
var DefaultPaths = map[string]map[string][]string{
	"nginx": {
		"access": {"/var/log/nginx/access.log*"},
		"error":  {"/var/log/nginx/error.log*"},
	},
	"mysql": {
		"slowlog":  {"/var/log/mysql/mysql-slow.log*"},
		"errorlog": {"/var/log/mysql/error.log*"},
	},
	"redis": {
		"slowlog": {"/var/log/redis/redis-slowlog.log*"},
		"general": {"/var/log/redis/redis.log*"},
	},
	"apache": {
		"access": {"/var/log/apache2/access.log*"},
		"error":  {"/var/log/apache2/error.log*"},
	},
	"mongodb": {
		"logs": {"/var/log/mongodb/mongodb.log"},
	},
	"kafka": {
		"log": {
			"/var/log/kafka/controller.log*",
			"/var/log/kafka/server.log*",
			"/var/log/kafka/state-change.log*",
			"/var/log/kafka/kafka-*.log*",
		},
	},
	"zookeeper": {
		"audit": {"/var/log/zookeeper/zookeeper_audit.log*"},
		"log":   {"/var/log/zookeeper/zookeeper.log*"},
	},
	"rabbitmq": {
		"audit": {"/var/log/rabbitmq/*.log*"},
	},
	"postgresql": {
		"log": {"/var/log/postgres/*.log*"},
	},
	"logstash": {
		"log":     {"/var/log/logstash/logstash.log*"},
		"slowlog": {"/var/log/logstash/logstash-slowlog.log*"},
	},
}

func GenerateFilebeatConfig(jsonData []byte, agentID string) (*FilebeatConfig, error) {
	var tasks []struct {
		ModuleName     string   `json:"module_name"`
		Mode           int      `json:"mode"`
		LogPath        []string `json:"log_path"`
		AccessLogPath  []string `json:"access_log_path"`
		ErrorLogPath   []string `json:"error_log_path"`
		SlowlogLogPath []string `json:"slowlog_log_path"`
	}

	// 解析 JSON 任务
	if err := json.Unmarshal(jsonData, &tasks); err != nil {
		return nil, err
	}

	config := &FilebeatConfig{
		Filebeat: struct {
			Modules []ModuleConfig `yaml:"modules,omitempty"`
			Inputs  []InputConfig  `yaml:"inputs,omitempty"`
		}{
			Modules: []ModuleConfig{}, // Add modules here if needed
			Inputs:  []InputConfig{},  // Add inputs here if needed
		},
		Fields: FieldsConfig{
			AgentID: agentID, // Assuming 'agentID' is a variable
		},
		Output: OutputConfig{
			Http: struct {
				Hosts []string `yaml:"hosts"`
				Path  string   `yaml:"path"`
			}{
				Hosts: []string{"http://172.27.220.125:6752"},
				Path:  "/receive/log",
			},
		},
	}

	// 用于存储生成的模块配置
	uniqueInputPaths := map[string]struct{}{}
	modulesExist := false

	// 遍历任务并根据 mode 生成配置
	for _, task := range tasks {
		if task.ModuleName == "" {
			if task.Mode == 1 && len(task.LogPath) > 0 {
				for _, logPath := range task.LogPath {
					if _, exists := uniqueInputPaths[logPath]; !exists {
						inputConfig := InputConfig{
							Type:    "log",
							Enabled: true,
							Paths:   []string{logPath},
						}
						config.Filebeat.Inputs = append(config.Filebeat.Inputs, inputConfig)
						uniqueInputPaths[logPath] = struct{}{}
					}
				}
			}
			continue
		}

		moduleConfig := ModuleConfig{
			Module: task.ModuleName,
		}
		modulesExist = true

		switch task.Mode {
		case 0:
			// 默认路径模式
			if defaultPaths, ok := DefaultPaths[task.ModuleName]; ok {
				if accessPaths, exists := defaultPaths["access"]; exists && len(accessPaths) > 0 {
					moduleConfig.Access = &struct {
						Enabled bool     `yaml:"enabled"`
						Paths   []string `yaml:"var.paths"`
					}{Enabled: true, Paths: accessPaths}
				}
				if errorPaths, exists := defaultPaths["error"]; exists && len(errorPaths) > 0 {
					moduleConfig.Error = &struct {
						Enabled bool     `yaml:"enabled"`
						Paths   []string `yaml:"var.paths"`
					}{Enabled: true, Paths: errorPaths}
				}
				if slowlogPaths, exists := defaultPaths["slowlog"]; exists && len(slowlogPaths) > 0 {
					moduleConfig.Slowlog = &struct {
						Enabled bool     `yaml:"enabled"`
						Paths   []string `yaml:"var.paths"`
					}{Enabled: true, Paths: slowlogPaths}
				}
			}
		case 1:
			// 自定义路径模式
			if len(task.LogPath) > 0 {
				for _, logPath := range task.LogPath {
					if _, exists := uniqueInputPaths[logPath]; !exists {
						inputConfig := InputConfig{
							Type:    "log",
							Enabled: true,
							Paths:   []string{logPath},
						}
						config.Filebeat.Inputs = append(config.Filebeat.Inputs, inputConfig)
						uniqueInputPaths[logPath] = struct{}{}
					}
				}
			}
		case 2:
			// 默认 + 自定义路径模式
			if len(task.AccessLogPath) > 0 {
				moduleConfig.Access = &struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{Enabled: true, Paths: task.AccessLogPath}
			}
			if len(task.ErrorLogPath) > 0 {
				moduleConfig.Error = &struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{Enabled: true, Paths: task.ErrorLogPath}
			}
			if len(task.SlowlogLogPath) > 0 {
				moduleConfig.Slowlog = &struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{Enabled: true, Paths: task.SlowlogLogPath}
			}
		}

		if modulesExist {
			config.Filebeat.Modules = append(config.Filebeat.Modules, moduleConfig)
		}
	}

	// 如果没有任何模块配置且输入配置为空，则删除 filebeat 里的 modules 字段
	if len(config.Filebeat.Modules) == 0 && len(config.Filebeat.Inputs) == 0 {
		config.Filebeat.Modules = nil
	}

	return config, nil
}

// 将配置写入 YAML 文件
func WriteConfigToFile(config *FilebeatConfig, filename string) error {
	// 将结构体转换为 YAML 格式
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %v", err)
	}

	// 写入文件
	err = ioutil.WriteFile(filename, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config to file: %v", err)
	}

	fmt.Println("Filebeat configuration written to", filename)
	return nil
}

// 获取 agentID 的函数
func GetAgentID(filePath string) (string, error) {
	// 读取指定路径的文件
	agentID, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("Error reading machine-id: %v", err)
	}

	// 清理 agentID 去除换行符和其他空白字符
	agentIDStr := string(agentID)
	agentIDStr = strings.TrimSpace(agentIDStr)

	return agentIDStr, nil
}
