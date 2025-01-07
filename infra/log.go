package infraLog

import (
	"fmt"
	"log"
	"os"
)

type LogConfig struct {
	LogFilePath string // 日志文件路径
	LogLevel    string // 日志级别 (INFO, ERROR, DEBUG)
}

type Logger struct {
	*log.Logger
}

var GlobalLog *Logger

func init() {
	// 设置默认日志配置
	logConfig := LogConfig{
		LogFilePath: "logSync.log", // 默认日志文件路径
		LogLevel:    "INFO",        // 默认日志级别
	}

	// 创建日志文件
	logFile, err := os.OpenFile(logConfig.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}

	// 创建自定义日志实例
	GlobalLog = &Logger{
		Logger: log.New(logFile, "[baseline]", log.LstdFlags|log.Lshortfile|log.LUTC),
	}

}

// LogLevel
const (
	INFO  = "INFO"
	ERROR = "ERROR"
	DEBUG = "DEBUG"
)

func (l *Logger) Log(level, message string) {
	if level == ERROR {
		l.Logger.SetPrefix("[ERROR] ")
	} else if level == DEBUG {
		l.Logger.SetPrefix("[DEBUG] ")
	} else {
		l.Logger.SetPrefix("[INFO] ")
	}

	l.Logger.Printf("%s - %s", level, message)
}

func (l *Logger) Error(message string) {
	l.Log(ERROR, message)
}

func (l *Logger) Info(message string) {
	l.Log(INFO, message)
}

func (l *Logger) Debug(message string) {
	l.Log(DEBUG, message)
}
