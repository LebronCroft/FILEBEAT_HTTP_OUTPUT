package send_bussiness

import (
	"bufio"
	"encoding/json"
	"fmt"
	infraLog "github.com/elastic/beats/v7/infra"
	"sort"

	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogEntryInfo represents a single log entry structure
type LogEntryInfo struct {
	BaselineId   int       `json:"baseline_id"`
	IP           string    `json:"ip"`
	AgentId      string    `json:"agent_id"`
	Datetime     time.Time `json:"datetime"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	Protocol     string    `json:"protocol"`
	StatusCode   int       `json:"status_code"`
	ResponseSize int       `json:"response_size"`
}

var (
	offsetFile  = "./offset.state" // 偏移量状态文件路径
	offsetMutex sync.Mutex         // 用于保护偏移量文件的并发安全
)

// OffsetMap 定义一个map，保存多个文件的偏移量
type OffsetMap map[string]int64

func LoadOffsets() (OffsetMap, error) {
	infraLog.GlobalLog.Info(fmt.Sprintf("Attempting to load offsets from file: %s", offsetFile))
	offsetMap := make(OffsetMap)

	file, err := os.Open(offsetFile)
	if err != nil {
		if os.IsNotExist(err) {
			infraLog.GlobalLog.Info("Offset state file does not exist, starting with empty offset map")
			return offsetMap, nil
		}
		return nil, fmt.Errorf("failed to open offset file: %v", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat offset file: %v", err)
	}
	if stat.Size() == 0 {
		infraLog.GlobalLog.Info("Offset file is empty, starting with empty offset map")
		return offsetMap, nil
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&offsetMap)
	if err != nil {
		if err.Error() == "EOF" {
			infraLog.GlobalLog.Info("Offset file is empty or malformed, starting with empty offset map")
			return offsetMap, nil
		}
		return nil, fmt.Errorf("failed to decode offset file: %v", err)
	}

	infraLog.GlobalLog.Info(fmt.Sprintf("Loaded offsets: %+v", offsetMap))
	return offsetMap, nil
}

// SaveOffsets 将所有文件的偏移量保存到状态文件
func SaveOffsets(offsetMap OffsetMap) error {
	offsetMutex.Lock()
	defer offsetMutex.Unlock()

	file, err := os.OpenFile(offsetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644) // 设置权限
	if err != nil {
		return fmt.Errorf("failed to create offset file: %v", err)
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(offsetMap)
	if err != nil {
		return fmt.Errorf("failed to encode offsets to file: %v", err)
	}

	infraLog.GlobalLog.Info(fmt.Sprintf("Saved offsets: %+v", offsetMap))
	return nil
}

// 最大行长度限制，避免过长的行造成问题
const maxLineLength = 1024 * 1024 // 1MB, 根据需要调整

func IncrementalRead(directoryPath string, baseFileName string) ([]LogEntryInfo, error) {
	infraLog.GlobalLog.Info(fmt.Sprintf("Starting incremental read for directory: %s", directoryPath))
	var result []LogEntryInfo
	const batchSize = 1000 // 每次返回的行数

	// 获取所有文件路径
	files, err := getLogFiles(directoryPath, baseFileName)
	if err != nil {
		return nil, err
	}

	// 加载偏移量
	offsetMap, err := LoadOffsets()
	if err != nil {
		return nil, err
	}

	// 遍历文件列表并逐个读取
	for _, filePath := range files {
		fileName := extractFileName(filePath)
		lastOffset, exists := offsetMap[fileName]
		if !exists {
			lastOffset = 0
		}

		// 尝试打开文件
		file, err := os.Open(filePath)
		if err != nil {
			// 记录日志并跳过文件
			infraLog.GlobalLog.Info(fmt.Sprintf("Could not open file %s: %v. Skipping file.", filePath, err))
			continue
		}
		defer file.Close()

		// 跳转到上次的偏移位置
		_, err = file.Seek(lastOffset, 0)
		if err != nil {
			return nil, fmt.Errorf("could not seek to last offset: %v", err)
		}

		// 使用 bufio.Reader 来处理超长的行
		reader := bufio.NewReader(file)

		// 逐行读取文件内容
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err.Error() != "EOF" {
				return nil, fmt.Errorf("error reading file: %v", err)
			}

			// 跳过空行
			if line == "" {
				break
			}

			// 如果行长度过长，跳过该行
			if len(line) > maxLineLength {
				infraLog.GlobalLog.Info(fmt.Sprintf("Skipping line: %s (line too long)", line))
				continue
			}

			// 清理可能存在的乱码字符或不可打印字符
			line = cleanUpLogLine(line)

			// 解析日志行
			entry, err := ParseLogLine(line)
			if err != nil {
				infraLog.GlobalLog.Info(fmt.Sprintf("Error parsing log line: %v", err))
				continue
			}
			result = append(result, *entry)

			// 每处理 1000 行就返回一次
			if len(result) >= batchSize {
				// 保存新的偏移量
				newOffset, _ := file.Seek(0, os.SEEK_CUR)
				offsetMap[fileName] = newOffset

				// 保存偏移量
				err = SaveOffsets(offsetMap)
				if err != nil {
					return nil, err
				}

				// 返回当前处理的 1000 行数据
				infraLog.GlobalLog.Info(fmt.Sprintf("Returning batch of %d log entries.", len(result)))
				return result, nil // 返回当前 1000 行
			}
		}

		// 保存新的偏移量
		newOffset, _ := file.Seek(0, os.SEEK_CUR)
		offsetMap[fileName] = newOffset

		// 保存偏移量
		err = SaveOffsets(offsetMap)
		if err != nil {
			return nil, err
		}
	}

	// 如果读取完所有文件仍有剩余的日志数据，则返回
	if len(result) > 0 {
		infraLog.GlobalLog.Info(fmt.Sprintf("Finished reading files in directory: %s, total entries read: %d", directoryPath, len(result)))
		return result, nil
	}

	// 如果没有数据返回
	return nil, nil
}

// cleanUpLogLine 用于清理不可打印的字符
func cleanUpLogLine(line string) string {
	var cleanedLine strings.Builder
	for _, ch := range line {
		// 只保留可打印字符，其他的都去掉
		if ch >= 32 && ch <= 126 {
			cleanedLine.WriteRune(ch)
		} else {
			// 将不可打印字符替换为空格或其他占位符
			cleanedLine.WriteRune(' ')
		}
	}
	return cleanedLine.String()
}

// getLogFiles 获取目录下所有符合命名规则的文件
func getLogFiles(directoryPath string, baseFileName string) ([]string, error) {
	var files []string

	// 读取目录
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只匹配特定文件名格式（如 logout_YYYYMMDD.json 或 logout_YYYYMMDD-1.json）
		if strings.HasPrefix(info.Name(), baseFileName) && strings.HasSuffix(info.Name(), ".ndjson") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no log files found in directory: %s", directoryPath)
	}

	// 按文件名排序（升序），确保先读取最早的文件
	sort.Strings(files)
	return files, nil
}

func ParseLogLine(line string) (*LogEntryInfo, error) {
	infraLog.GlobalLog.Debug(fmt.Sprintf("Parsing line: %s", line))
	var entry map[string]interface{}

	// 检查空行或非 JSON 格式的日志
	if line == "" {
		infraLog.GlobalLog.Debug("Skipping empty line")
		return nil, nil
	}

	// 尝试解析 JSON
	err := json.Unmarshal([]byte(line), &entry)
	if err != nil {
		infraLog.GlobalLog.Info(fmt.Sprintf("Error unmarshaling line: %s, error: %v", line, err))
		return nil, fmt.Errorf("could not unmarshal line: %v", err)
	}

	// 成功解析 JSON 后，创建 LogEntryInfo
	return CreateLogEntryInfo(entry)
}

func CreateLogEntryInfo(entry map[string]interface{}) (*LogEntryInfo, error) {
	message, ok := entry["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message field missing or not a string")
	}

	logPattern := `([\d\.]+) \S+ \S+ \[([^\]]+)] \"(\S+) (.*?) HTTP/(\S+)\" (\d+) (\d+)`
	reg := regexp.MustCompile(logPattern)
	matches := reg.FindStringSubmatch(message)
	if len(matches) < 7 {
		return nil, fmt.Errorf("unable to parse log message with expected format")
	}

	clientIP := matches[1]
	timestampStr := matches[2]
	datetime, err := time.Parse("02/Jan/2006:15:04:05 -0700", timestampStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse datetime: %v", err)
	}

	method := matches[3]
	requestURL := matches[4]
	httpVersion := matches[5]
	statusCode, err := strconv.Atoi(matches[6])
	if err != nil {
		return nil, fmt.Errorf("invalid status code value: %v", err)
	}

	return &LogEntryInfo{
		IP:         clientIP,
		Datetime:   datetime,
		Method:     method,
		URL:        requestURL,
		Protocol:   httpVersion,
		StatusCode: statusCode,
	}, nil
}

func extractFileName(filePath string) string {
	// 从路径中提取文件名（如 log_output-20250106-159.ndjson）
	fileName := filepath.Base(filePath)

	// 使用正则表达式匹配文件名格式 log_output-YYYYMMDD-<version>.ndjson 或 log_output-YYYYMMDD.ndjson
	re := regexp.MustCompile(`^log_output-\d{8}(-\d+)?\.ndjson$`) // 匹配 log_output-YYYYMMDD 或 log_output-YYYYMMDD-<数字>.ndjson
	match := re.FindString(fileName)
	if match == "" {
		infraLog.GlobalLog.Error(fmt.Sprintf("Invalid file name format: %v", filePath))
		return ""
	}

	// 去除文件扩展名 (.ndjson)
	return strings.TrimSuffix(match, ".ndjson")
}
