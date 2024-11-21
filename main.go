package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/wukongcloud/mqtt-performace-test/app/throughput"
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
	//connect.ConnectTest(config)
	throughput.ThroughputTest(config)
}
