package infraLog

import (
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
	"time"
)

type LogConfig struct {
	LogFilePath    string // 日志文件路径
	MaxSize        int    // 每个日志文件最大尺寸（单位：MB）
	MaxBackups     int    // 最多保留的旧日志文件个数
	MaxAge         int    // 日志保留最大天数（单位：天）
	Compress       bool   // 是否压缩旧日志文件
	LogLevel       string // 日志级别 (INFO, WARN, DEBUG, ERROR)
	RotationPeriod int    // 日志轮转周期（单位：秒）
	ModuleName     string // 模块名
}

type Logger struct {
	*log.Logger
}

var (
	GlobalLog *Logger
	logFiles  = map[string]*lumberjack.Logger{}
)

func init() {
	logConfig := LogConfig{
		LogFilePath: "./logs/",             // 默认日志目录
		MaxSize:     100,                  // 每个日志文件最大 100MB
		MaxBackups:  3,                    // 最多保留 3 个旧日志文件
		MaxAge:      30,                   // 日志最多保存 30 天
		Compress:    true,                 // 是否压缩旧日志文件
		LogLevel:    "INFO",               // 默认日志级别
		ModuleName:  "plugin_logsync_001", // 模块名
	}

	// 创建日志目录，如果不存在
	if err := os.MkdirAll(logConfig.LogFilePath, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create log directory: %v", err))
	}

	// 按级别创建不同日志文件
	logLevels := []string{"info", "warn", "debug", "error", "fatal"}
	for _, level := range logLevels {
		// 使用日期（YYYY-MM-DD）生成日志文件名
		logFileName := fmt.Sprintf("%s-%s.log", logConfig.ModuleName, time.Now().Format("2006-01-02"))
		logFile := &lumberjack.Logger{
			Filename:   filepath.Join(logConfig.LogFilePath, logFileName),
			MaxSize:    logConfig.MaxSize,
			MaxBackups: logConfig.MaxBackups,
			MaxAge:     logConfig.MaxAge,
			Compress:   logConfig.Compress,
		}
		logFiles[level] = logFile
	}

	// 默认全局日志写入到 info 日志中
	GlobalLog = &Logger{
		Logger: log.New(logFiles["info"], "", log.LstdFlags|log.Lshortfile|log.LUTC),
	}
}

// 日志记录函数
func (l *Logger) Log(level, message string) {
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z") // ISO 8601 格式
	writer, ok := logFiles[level]
	if !ok {
		writer = logFiles["info"] // 默认写到 info 日志中
	}

	logger := log.New(writer, "", log.LstdFlags|log.Lshortfile|log.LUTC)
	logger.Printf("%s - %s - %s", timestamp, level, message)
}

// 日志快捷方法
func (l *Logger) Error(message string) { l.Log("error", message) }
func (l *Logger) Info(message string)  { l.Log("info", message) }
func (l *Logger) Debug(message string) { l.Log("debug", message) }
func (l *Logger) Warn(message string)  { l.Log("warn", message) }
func (l *Logger) Fatal(message string) { l.Log("fatal", message) }
