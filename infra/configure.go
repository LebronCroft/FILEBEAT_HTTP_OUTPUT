package infraLog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

func InitGlobalLogger(logConfig LogConfig) {
	if logConfig.LogFilePath == "" {
		logConfig.LogFilePath = "./logs/"
	}
	if logConfig.MaxSize <= 0 {
		logConfig.MaxSize = 100
	}
	if logConfig.MaxBackups <= 0 {
		logConfig.MaxBackups = 3
	}
	if logConfig.MaxAge <= 0 {
		logConfig.MaxAge = 30
	}
	if logConfig.ModuleName == "" {
		logConfig.ModuleName = "filebeat-http-output"
	}

	if err := os.MkdirAll(logConfig.LogFilePath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create log directory: %v", err))
	}

	logFiles = map[string]*lumberjack.Logger{}
	for _, level := range []string{"info", "warn", "debug", "error", "fatal"} {
		logFileName := fmt.Sprintf("%s-%s.log", logConfig.ModuleName, time.Now().Format("2006-01-02"))
		logFiles[level] = &lumberjack.Logger{
			Filename:   filepath.Join(logConfig.LogFilePath, logFileName),
			MaxSize:    logConfig.MaxSize,
			MaxBackups: logConfig.MaxBackups,
			MaxAge:     logConfig.MaxAge,
			Compress:   logConfig.Compress,
		}
	}

	GlobalLog = &Logger{
		Logger: log.New(logFiles["info"], "", log.LstdFlags|log.Lshortfile|log.LUTC),
	}
}
