package main

import (
	"fmt"
	"os"
	"os/exec"

	filebeatConfig "github.com/fufuok/beats-http-output/config"
	"github.com/fufuok/beats-http-output/enum"
	infraLog "github.com/fufuok/beats-http-output/infra"
	_ "github.com/fufuok/beats-http-output/libbeat/outputs/http"
	"github.com/fufuok/beats-http-output/script"
)

var cmd *exec.Cmd

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	configFile := wd + enum.FileBeatConfigFile
	cfg, err := filebeatConfig.LoadRuntimeConfig(configFile)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to load runtime config: %v", err))
		return
	}

	infraLog.InitGlobalLogger(infraLog.LogConfig{
		LogFilePath: cfg.Logging.Path,
		MaxSize:     cfg.Logging.MaxSize,
		MaxBackups:  cfg.Logging.MaxBackups,
		MaxAge:      cfg.Logging.MaxAge,
		Compress:    cfg.Logging.Compress,
		ModuleName:  cfg.Logging.ModuleName,
	})

	manager, err := script.NewManager(cfg)
	if err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to create script manager: %v", err))
		return
	}
	defer manager.Stop()

	defer func() {
		if cmd != nil && cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to kill filebeat process: %v", err))
			} else {
				infraLog.GlobalLog.Info("Filebeat process killed successfully.")
			}
		}
	}()

	// Start Filebeat
	infraLog.GlobalLog.Info("Starting Filebeat with config file: " + configFile)
	cmd = startFileBeat(configFile)

	if err := manager.Start(); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start scripts: %v", err))
		return
	}

	select {}
}

func startFileBeat(configFile string) *exec.Cmd {
	enum.Mu.Lock()
	defer enum.Mu.Unlock()

	infraLog.GlobalLog.Info("Starting Filebeat with config file: " + configFile)

	// Check if filebeatexc exists
	if _, err := os.Stat("./filebeatexc"); os.IsNotExist(err) {
		infraLog.GlobalLog.Error("filebeatexc not found in the current directory")
		return nil
	}

	// Ensure that filebeatexc has executable permissions
	if err := os.Chmod("./filebeatexc", 0755); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to set executable permissions on filebeatexc: %v", err))
		return nil
	}

	if _, err := os.Stat("./filebeatexc"); os.IsNotExist(err) {
		infraLog.GlobalLog.Error("filebeatexc not found in the current directory")
		return nil
	}

	cmd = exec.Command("./filebeatexc", "-e", "-c", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Failed to start filebeat: %v", err))
		return nil
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat process exited with error: %v", err))
		}
		infraLog.GlobalLog.Info("Filebeat process exited. Attempting to restart...")
		startFileBeat(configFile)
	}()

	infraLog.GlobalLog.Info("Filebeat process started successfully.")
	return cmd
}
