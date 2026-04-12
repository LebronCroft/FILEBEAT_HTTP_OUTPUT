package main

import (
	"fmt"
	filebeatConfig "github.com/fufuok/beats-http-output/config"
	"github.com/fufuok/beats-http-output/enum"
	infraLog "github.com/fufuok/beats-http-output/infra"
	_ "github.com/fufuok/beats-http-output/libbeat/outputs/http"
	"github.com/fufuok/beats-http-output/send_bussiness"
	"os"
	"os/exec"
)

var cmd *exec.Cmd

func main() {
	defer func() {
		if cmd != nil && cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("Failed to kill filebeat process: %v", err))
			} else {
				infraLog.GlobalLog.Info("Filebeat process killed successfully.")
			}
		}
		println("main exit")
	}()

	infraLog.GlobalLog.Info("Filebeat config generated successfully.")

	// Start Filebeat
	infraLog.GlobalLog.Info("Starting Filebeat with config file: " + configFile)
	cmd = startFileBeat(configFile)

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
