package main

import (
	"encoding/json"
	"github.com/wukongcloud/mqtt-performace-test/app/throughput"
	"log"
	"os"

	"github.com/wukongcloud/mqtt-performace-test/models"
)

func main() {
	// Load configuration
	configFile := "config.json"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}
	var config models.Config
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Invalid config file: %v", err)
	}

	if err := os.MkdirAll(config.ReportDir, 0755); err != nil {
		log.Fatalf("Failed to create report dir %s: %v", config.ReportDir, err)
	}

	//connect.RunConnectTest(config)
	//eventLatency.EventLatencyTest(config)

	throughput.RunThroughputTest(config, 1)
}
