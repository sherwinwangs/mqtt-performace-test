package throughput

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

// Perform the throughput test by connecting clients in batches
func ThroughputTest(config models.Config) {
	var connectWg sync.WaitGroup // Synchronize client connections
	var messageWg sync.WaitGroup // Synchronize message sending
	messageChan := make(chan models.MessageMetric, config.MaxConnections)

	totalSuccessfulClients := 0
	totalFailedClients := 0
	totalConnectionTime := time.Duration(0)

	var connectMetrics []models.ConnectMetric

	// Shared end time for all clients
	testEndTime := time.Now().Add(config.TestDuration.Duration)

	startTime := time.Now() // Record test start time

	// Iterate through batches
	for batch := 1; totalSuccessfulClients+totalFailedClients < config.MaxConnections; batch++ {
		currentBatch := config.IncrementStep
		if totalSuccessfulClients+totalFailedClients+currentBatch > config.MaxConnections {
			currentBatch = config.MaxConnections - (totalSuccessfulClients + totalFailedClients)
		}

		// Connection metrics for this batch
		batchConnectTimes := []time.Duration{}
		batchSuccess := 0
		batchFail := 0

		// Start clients in the current batch
		for i := 0; i < currentBatch; i++ {
			clientID := fmt.Sprintf("LE40B8BBML1%06d", totalSuccessfulClients+totalFailedClients+i+1)
			connectWg.Add(1)

			go func(id string) {
				defer connectWg.Done()

				startTime := time.Now()
				caFile := filepath.Join(config.ClientFolderPath, "ca.crt")
				certFile := filepath.Join(config.ClientFolderPath, id, "client.chain.crt")
				keyFile := filepath.Join(config.ClientFolderPath, id, "client.key")

				tlsConfig, err := utils.NewTLSConfig(caFile, certFile, keyFile)
				if err != nil {
					log.Printf("Client %s failed to load TLS config: %v", id, err)
					batchFail++
					return
				}

				opts := mqtt.NewClientOptions().AddBroker(config.BrokerAddress).SetClientID(id).SetTLSConfig(tlsConfig)
				client := mqtt.NewClient(opts)

				if token := client.Connect(); token.WaitTimeout(config.ConnectTimeout.Duration) && token.Error() != nil {
					log.Printf("Client %s failed to connect: %v", id, token.Error())
					batchFail++
					return
				}

				connectionTime := time.Since(startTime)
				batchConnectTimes = append(batchConnectTimes, connectionTime)
				totalConnectionTime += connectionTime // Accumulate total connection time
				batchSuccess++

				// Start sending messages
				messageWg.Add(1)
				go utils.SendMessages(client, id, fmt.Sprintf("test/to/%s", id), utils.GenerateMessage(config.MessageSize), testEndTime, &config, messageChan, &messageWg)
			}(clientID)
		}

		connectWg.Wait()
		totalSuccessfulClients += batchSuccess
		totalFailedClients += batchFail

		// Calculate batch metrics
		var avgConnTime, minConnTime, maxConnTime time.Duration
		if len(batchConnectTimes) > 0 {
			totalBatchTime := time.Duration(0)
			minConnTime = batchConnectTimes[0]
			maxConnTime = batchConnectTimes[0]

			for _, connTime := range batchConnectTimes {
				totalBatchTime += connTime
				if connTime < minConnTime {
					minConnTime = connTime
				}
				if connTime > maxConnTime {
					maxConnTime = connTime
				}
			}
			avgConnTime = totalBatchTime / time.Duration(len(batchConnectTimes))
		}

		connectMetrics = append(connectMetrics, models.ConnectMetric{
			BatchNumber:           batch,
			SuccessfulConnections: batchSuccess,
			FailedConnections:     batchFail,
			AverageConnectionTime: avgConnTime,
			MinConnectionTime:     minConnTime,
			MaxConnectionTime:     maxConnTime,
		})

		//		log.Printf("Batch %d completed: Success: %d, Fail: %d", batch, batchSuccess, batchFail)
		log.Printf("Batch %d: Success: %d, Fail: %d, Avg: %v, Min: %v, Max: %v",
			batch, batchSuccess, batchFail, avgConnTime, minConnTime, maxConnTime)
	}

	// Wait for all message goroutines to finish after the test duration
	messageWg.Wait()
	close(messageChan)

	// Collect message metrics
	totalMessagesSent := 0
	totalMessagesFailed := 0
	//var messageMetrics []models.MessageMetric

	for metrics := range messageChan {
		totalMessagesSent += metrics.MessagesSent
		totalMessagesFailed += metrics.MessagesFailed
		//messageMetrics = append(messageMetrics, metrics)
	}

	endTime := time.Now() // Record test end time

	// Calculate overall average connection time
	overallAvgConnTime := time.Duration(0)
	if totalSuccessfulClients > 0 {
		overallAvgConnTime = totalConnectionTime / time.Duration(totalSuccessfulClients)
	}
	// Calculate average connections per second
	testDuration := endTime.Sub(startTime).Seconds()
	avgConnectionsPerSecond := float64(totalSuccessfulClients) / testDuration

	// Calculate throughput
	duration := config.TestDuration.Duration.Seconds()
	throughput := float64(totalMessagesSent) / duration

	// Prepare the report
	summary := models.SummaryData{
		TotalConnectionsAttempted:    totalSuccessfulClients + totalFailedClients,
		TotalSuccessfulConnections:   totalSuccessfulClients,
		TotalFailedConnections:       totalFailedClients,
		OverallAverageConnectionTime: overallAvgConnTime,
		AverageConnectionsPerSecond:  avgConnectionsPerSecond,
		StartTime:                    startTime,
		EndTime:                      endTime,
		ConnectMetrics:               connectMetrics,
		TotalMessagesSent:            totalMessagesSent,
		TotalMessagesFailed:          totalMessagesFailed,
		OverallThroughput:            throughput,
		//MessageMetrics:               messageMetrics,
	}

	// Write report to file
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

	log.Printf("Throughput test completed. Report saved to %s", config.ReportFile)
}
