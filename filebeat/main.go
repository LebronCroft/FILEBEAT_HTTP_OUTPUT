package main

import (
	"fmt"
	"github.com/elastic/beats/v7/filebeat/cmd"
	inputs "github.com/elastic/beats/v7/filebeat/input/default-inputs"
	infraLog "github.com/elastic/beats/v7/infra"
	"os"
	_ "time/tzdata" // for timezone handling
)


// main
func main() {
	if err := cmd.Filebeat(inputs.Init, cmd.FilebeatSettings("")).Execute(); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("Filebeat failed to start: %v", err))
		os.Exit(1)
	}

}
