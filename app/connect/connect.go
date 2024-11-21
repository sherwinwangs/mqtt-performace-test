package connect

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wukongcloud/mqtt-performace-test/models"
	"github.com/wukongcloud/mqtt-performace-test/utils"
)

func connectClient(clientID, brokerAddress, clientFolderPath string, timeout time.Duration, connDataChan chan<- models.ConnectionData) {
	caFile := filepath.Join(clientFolderPath, "ca.crt")
	certFile := filepath.Join(clientFolderPath, clientID, "client.chain.crt")
	keyFile := filepath.Join(clientFolderPath, clientID, "client.key")

	tlsConfig, err := utils.NewTLSConfig(caFile, certFile, keyFile)
	if err != nil {
		connDataChan <- models.ConnectionData{ClientID: clientID, ConnectionError: true}
		log.Printf("Client %s failed to load TLS config: %v", clientID, err)
		return
	}

	opts := mqtt.NewClientOptions().AddBroker(brokerAddress).SetClientID(clientID).SetTLSConfig(tlsConfig)
	client := mqtt.NewClient(opts)

	startTime := time.Now()
	token := client.Connect()

	// Wait with timeout
	if ok := token.WaitTimeout(timeout); !ok || token.Error() != nil {
		connDataChan <- models.ConnectionData{ClientID: clientID, ConnectionError: true}
		//log.Printf("Client %s failed to connect within timeout: %v", clientID, token.Error())
		return
	}

	connTime := time.Since(startTime)
	connDataChan <- models.ConnectionData{ClientID: clientID, ConnectionTime: connTime}
	//log.Printf("Client %s connected successfully in %v", clientID, connTime)
}

func ConnectTest(config models.Config) {
	connDataChan := make(chan models.ConnectionData, config.MaxConnections)
	var wg sync.WaitGroup

	totalConnections := 0
	successfulConnections := 0
	failedConnections := 0
	var totalConnTime time.Duration
	var batchMetrics []models.ConnectMetric

	startTime := time.Now() // Record test start time
	for batch := 1; totalConnections < config.MaxConnections; batch++ {
		currentBatch := config.IncrementStep
		if totalConnections+currentBatch > config.MaxConnections {
			currentBatch = config.MaxConnections - totalConnections
		}

		// Start a batch of connections
		for i := 0; i < currentBatch; i++ {
			//clientID := fmt.Sprintf("Client%06d", totalConnections+i+1)
			clientID := fmt.Sprintf("LE40B8BBML1%06d", totalConnections+i+1)
			wg.Add(1)
			go func() {
				defer wg.Done()
				connectClient(clientID, config.BrokerAddress, config.ClientFolderPath, config.ConnectTimeout.Duration, connDataChan)
			}()
		}

		wg.Wait()

		// Process connection data
		batchSuccess := 0
		batchFail := 0
		var batchConnTimes []time.Duration
		for i := 0; i < currentBatch; i++ {
			data := <-connDataChan
			totalConnections++
			if data.ConnectionError {
				failedConnections++
				batchFail++
			} else {
				successfulConnections++
				totalConnTime += data.ConnectionTime
				batchSuccess++
				batchConnTimes = append(batchConnTimes, data.ConnectionTime)
			}
		}

		// Calculate batch metrics
		var avgConnTime, minConnTime, maxConnTime time.Duration
		if len(batchConnTimes) > 0 {
			minConnTime = batchConnTimes[0]
			maxConnTime = batchConnTimes[0]
			totalBatchTime := time.Duration(0)
			for _, t := range batchConnTimes {
				totalBatchTime += t
				if t < minConnTime {
					minConnTime = t
				}
				if t > maxConnTime {
					maxConnTime = t
				}
			}
			avgConnTime = totalBatchTime / time.Duration(len(batchConnTimes))
		}

		batchMetrics = append(batchMetrics, models.ConnectMetric{
			BatchNumber:           batch,
			SuccessfulConnections: batchSuccess,
			FailedConnections:     batchFail,
			AverageConnectionTime: avgConnTime,
			MinConnectionTime:     minConnTime,
			MaxConnectionTime:     maxConnTime,
		})

		log.Printf("Batch %d: Success: %d, Fail: %d, Avg: %v, Min: %v, Max: %v",
			batch, batchSuccess, batchFail, avgConnTime, minConnTime, maxConnTime)
	}

	endTime := time.Now() // Record test end time

	// Calculate overall average connection time
	var overallAvgConnTime time.Duration
	if successfulConnections > 0 {
		overallAvgConnTime = totalConnTime / time.Duration(successfulConnections)
	}

	// Calculate average connections per second
	testDuration := endTime.Sub(startTime).Seconds()
	avgConnectionsPerSecond := float64(successfulConnections) / testDuration

	// Prepare summary data
	summary := models.SummaryData{
		TotalConnectionsAttempted:    totalConnections,
		TotalSuccessfulConnections:   successfulConnections,
		TotalFailedConnections:       failedConnections,
		OverallAverageConnectionTime: overallAvgConnTime,
		AverageConnectionsPerSecond:  avgConnectionsPerSecond,
		StartTime:                    startTime,
		EndTime:                      endTime,
		ConnectMetrics:               batchMetrics,
	}

	// Write results to report file
	reportFile, err := os.Create(config.ReportFile)
	if err != nil {
		log.Fatalf("Failed to create report file: %v", err)
	}
	defer reportFile.Close()

	reportJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize report data: %v", err)
	}
	reportFile.Write(reportJSON)

	log.Printf("Connection scalability test completed. Report saved to %s", config.ReportFile)
}
