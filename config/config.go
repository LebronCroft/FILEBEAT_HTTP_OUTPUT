package filebeatConfig

import (
	"fmt"
	infraLog "github.com/fufuok/beats-http-output/infra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"regexp"
	"strings"
)

var fileBeatConfigName = "filebeat.yml"

type ModuleConfig struct {
	ModuleName    string   `json:"module_name"`
	Mode          int      `json:"mode"`
	AccessLogPath []string `json:"access_log_path"`
	ErrorLogPath  []string `json:"error_log_path"`
	SysLogPath    []string `json:"sys_log_path"`
}

// 定义结构体
type FilebeatConfig struct {
	Filebeat struct {
		Config struct {
			Modules struct {
				Enabled bool   `yaml:"enabled"`
				Path    string `yaml:"path"`
				Reload  struct {
					Enabled bool   `yaml:"enabled"`
					Period  string `yaml:"period"`
				} `yaml:"reload"`
			} `yaml:"modules"`
		} `yaml:"config"`
	} `yaml:"filebeat"`
	Fields struct {
		AgentID string `yaml:"agentID"`
	} `yaml:"fields"`
	Output struct {
		HTTP struct {
			Hosts []string `yaml:"hosts"`
			Path  string   `yaml:"path"`
		} `yaml:"http"`
	} `yaml:"output"`
}

func UpdateFilebeatConfig(jsonInput string, filebeatFilePath string) error {
	infraLog.GlobalLog.Info("Starting UpdateFilebeatConfig...")

	var configs []ModuleConfig
	infraLog.GlobalLog.Debug(fmt.Sprintf("Received JSON input: %s", jsonInput))

	infraLog.GlobalLog.Debug("Parsing JSON input...")
	err := json.Unmarshal([]byte(jsonInput), &configs)
	if err != nil {
		fmt.Println("Failed to parse JSON:", err)
		return err
	}
	// 修复 JSON 数据中的错误格式
	fixedConfigs := fixInvalidJson(configs)

	// 打印修复后的配置
	fmt.Println("Fixed JSON configuration:")
	for _, config := range fixedConfigs {
		fmt.Printf("%+v\n", config)
	}

	infraLog.GlobalLog.Debug(fmt.Sprintf("Parsed JSON successfully: %+v", configs))

	// Initialize the filebeat.yml content
	var filebeatConfigContent strings.Builder

	// Process each module configuration
	for _, config := range configs {
		infraLog.GlobalLog.Info(fmt.Sprintf("Processing config: module_name=%s, mode=%d", config.ModuleName, config.Mode))

		if config.Mode == 0 {
			// Mode 0: Generate configuration based on module file
			moduleFilePath := filebeatFilePath + fmt.Sprintf("%s.yml", config.ModuleName)
			infraLog.GlobalLog.Info(fmt.Sprintf("Detected mode=0. Attempting to read file: %s", moduleFilePath))

			moduleContent, err := ioutil.ReadFile(moduleFilePath)
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to read %s: %v", moduleFilePath, err))
				return fmt.Errorf("failed to read %s: %w", moduleFilePath, err)
			}
			infraLog.GlobalLog.Debug(fmt.Sprintf("Successfully read content from %s", moduleFilePath))

			filebeatConfigContent.WriteString(string(moduleContent))
			filebeatConfigContent.WriteString("\n")
		}

		if config.Mode == 1 {
			// Mode 2: Generate custom configuration based on custom paths
			infraLog.GlobalLog.Info("Detected mode=2. Using custom paths for access and error logs.")
			customConfig := generateCustomFilebeatConfig(config)
			infraLog.GlobalLog.Debug("Generated custom filebeat configuration:\n" + customConfig)

			filebeatConfigContent.WriteString(customConfig)
			filebeatConfigContent.WriteString("\n")
		}
	}

	// Write the generated configuration to the filebeat.yml
	err = ioutil.WriteFile(filebeatFilePath+fileBeatConfigName, []byte(filebeatConfigContent.String()), 0644)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to write to %s: %v", filebeatFilePath+fileBeatConfigName, err))
		return fmt.Errorf("failed to write to %s: %w", filebeatFilePath+fileBeatConfigName, err)
	}

	infraLog.GlobalLog.Info("filebeat.yml has been updated successfully.")
	return nil
}

// 清洗 JSON 字符串中的错误格式
func cleanModuleName(moduleName string) string {
	// 去除多余的引号和转义字符
	re := regexp.MustCompile(`"([^"]+)"`)
	return re.ReplaceAllString(moduleName, `$1`)
}

func cleanSysLogPath(sysLogPaths []string) []string {
	// 清理 sys_log_path 中的错误转义字符
	for i, path := range sysLogPaths {
		// 处理每个路径的错误格式
		sysLogPaths[i] = cleanModuleName(path)
	}
	return sysLogPaths
}

// 修复 JSON 数据中的错误格式
func fixInvalidJson(configs []ModuleConfig) []ModuleConfig {
	for i, config := range configs {
		// 修复 module_name 错误格式
		config.ModuleName = cleanModuleName(config.ModuleName)

		// 如果有 sys_log_path，修复其中的格式问题
		if config.SysLogPath != nil {
			config.SysLogPath = cleanSysLogPath(config.SysLogPath)
		}

		// 更新修复后的配置
		configs[i] = config
	}
	return configs
}

// generateCustomFilebeatConfig generates custom filebeat.yml content based on the config
func generateCustomFilebeatConfig(config ModuleConfig) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("- module: %s\n", config.ModuleName))

	// Access log configuration
	if len(config.AccessLogPath) > 0 {
		builder.WriteString("  access:\n")
		builder.WriteString("    enabled: true\n")
		builder.WriteString("    var:\n")
		builder.WriteString("      paths:\n")
		for _, path := range config.AccessLogPath {
			builder.WriteString(fmt.Sprintf("        - %s\n", path))
		}
	}

	// Error log configuration
	if len(config.ErrorLogPath) > 0 {
		builder.WriteString("  error:\n")
		builder.WriteString("    enabled: true\n")
		builder.WriteString("    var:\n")
		builder.WriteString("      paths:\n")
		for _, path := range config.ErrorLogPath {
			builder.WriteString(fmt.Sprintf("        - %s\n", path))
		}
	}

	// Syslog configuration (for system module)
	if len(config.SysLogPath) > 0 {
		builder.WriteString("  syslog:\n")
		builder.WriteString("    enabled: true\n")
		builder.WriteString("    var:\n")
		builder.WriteString("      paths:\n")
		for _, path := range config.SysLogPath {
			builder.WriteString(fmt.Sprintf("        - %s\n", path))
		}
	}

	// Add mysql slowlog (if applicable)
	if config.ModuleName == "mysql" {
		builder.WriteString("  slowlog:\n")
		builder.WriteString("    enabled: true\n")
		builder.WriteString("    var:\n")
		builder.WriteString("      paths:\n")
		builder.WriteString("        - /var/log/mysql/slowlog.log*\n")
	}

	return builder.String()
}

// 修改配置文件中的 agentID
func UpdateAgentIDInConfigFile(configFile string, newAgentID string) error {
	// 读取原始配置文件内容
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	// 解析 YAML 内容到结构体
	var config FilebeatConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("Error parsing YAML: %v", err)
	}
	// 确保 agentID 为字符串类型
	// 强制转换为字符串并更新
	config.Fields.AgentID = fmt.Sprintf("%v", newAgentID)
	// 将更新后的结构体重新编码为 YAML 格式
	updatedData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("Error marshaling updated config: %v", err)
	}

	// 将更新后的内容写回到配置文件
	err = ioutil.WriteFile(configFile, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("Error writing updated config file: %v", err)
	}

	return nil
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
