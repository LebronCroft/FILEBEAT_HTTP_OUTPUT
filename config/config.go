package config

import (
	"encoding/json"
	"fmt"
	infraLog "github.com/fufuok/beats-http-output/infra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
)

// FilebeatConfig 存储 Filebeat 配置结构
type FilebeatConfig struct {
	Filebeat struct {
		Modules []ModuleConfig `yaml:"modules"`
		Fields  FieldsConfig   `yaml:"fields"`
	} `yaml:"filebeat"`
}

// ModuleConfig 存储每个模块的配置
type ModuleConfig struct {
	Module string       `yaml:"module"`
	MySQL  *MySQLConfig `yaml:"mysql,omitempty"`
	Redis  *RedisConfig `yaml:"redis,omitempty"`
	Nginx  *NginxConfig `yaml:"nginx,omitempty"`
}

// MySQLConfig 存储 MySQL 配置
type MySQLConfig struct {
	Slowlog struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"slowlog"`
	Errorlog struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"errorlog"`
}

// RedisConfig 存储 Redis 配置
type RedisConfig struct {
	Slowlog struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"slowlog"`
	GeneralLog struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"general_log"`
}

// NginxConfig 存储 Nginx 配置
type NginxConfig struct {
	Access struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"access"`
	Error struct {
		Enabled bool     `yaml:"enabled"`
		Paths   []string `yaml:"var.paths"`
	} `yaml:"error"`
}

// FieldsConfig 存储全局字段配置
type FieldsConfig struct {
	AgentID string `yaml:"agentID"`
}

// NewFilebeatConfig 根据需要采集的日志类型生成配置
func NewFilebeatConfig(agentID, logType string) *FilebeatConfig {
	config := &FilebeatConfig{
		Filebeat: struct {
			Modules []ModuleConfig `yaml:"modules"`
			Fields  FieldsConfig   `yaml:"fields"`
		}{
			Fields: FieldsConfig{
				AgentID: agentID,
			},
		},
	}

	// 根据传入的 logType 加载不同的日志配置
	switch logType {
	case "nginx":
		config.Filebeat.Modules = append(config.Filebeat.Modules, ModuleConfig{
			Module: "nginx",
			Nginx: &NginxConfig{
				Access: struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{
					Enabled: true,
					Paths:   []string{"/var/log/nginx/access.log*"},
				},
				Error: struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{
					Enabled: true,
					Paths:   []string{"/var/log/nginx/error.log*"},
				},
			},
		})
	case "mysql":
		config.Filebeat.Modules = append(config.Filebeat.Modules, ModuleConfig{
			Module: "mysql",
			MySQL: &MySQLConfig{
				Slowlog: struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{
					Enabled: true,
					Paths:   []string{"/var/log/mysql/mysql-slow.log*"},
				},
				Errorlog: struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{
					Enabled: true,
					Paths:   []string{"/var/log/mysql/error.log*"},
				},
			},
		})
	case "redis":
		config.Filebeat.Modules = append(config.Filebeat.Modules, ModuleConfig{
			Module: "redis",
			Redis: &RedisConfig{
				Slowlog: struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{
					Enabled: true,
					Paths:   []string{"/var/log/redis/redis-slowlog.log*"},
				},
				GeneralLog: struct {
					Enabled bool     `yaml:"enabled"`
					Paths   []string `yaml:"var.paths"`
				}{
					Enabled: true,
					Paths:   []string{"/var/log/redis/redis.log*"},
				},
			},
		})
	default:
		// 默认情况下，不启用任何日志模块
		fmt.Println("No valid log type provided.")
	}

	return config
}

// WriteToFile 保存配置到 filebeat.yml
func (config *FilebeatConfig) WriteToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(config)
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}
	return nil
}

// ParseTaskData 解析任务中的 data 字段，提取所有键值对
func ParseTaskData(data interface{}) (map[string]string, error) {
	infraLog.GlobalLog.Info(fmt.Sprintf("Parsing task data %s", data))
	result := make(map[string]string)

	switch v := data.(type) {
	case string:
		// 解析 JSON 字符串为 map
		var dataMap map[string]interface{}
		err := json.Unmarshal([]byte(v), &dataMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON string: %v", err)
		}

		// 遍历 dataMap 并将键值对转换为字符串形式存入 result
		for key, value := range dataMap {
			if strValue, ok := value.(string); ok {
				result[key] = strValue
			} else {
				return nil, fmt.Errorf("value for key '%s' is not a string", key)
			}
		}

	case map[string]interface{}:
		// 如果 data 本身是一个 map，直接遍历并提取键值对
		for key, value := range v {
			if strValue, ok := value.(string); ok {
				result[key] = strValue
			} else {
				return nil, fmt.Errorf("value for key '%s' is not a string", key)
			}
		}

	default:
		return nil, fmt.Errorf("failed to parse task data: expected string or map, got %T", v)
	}

	return result, nil
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
