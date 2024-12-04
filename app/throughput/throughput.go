package throughput

import (
	"encoding/binary"
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

// subscribeToTopic receives messages and calculates latencies.
func subscribeToTopic(clientID, topic, brokerAddress, clientFolderPath string, qos int, testDuration time.Duration, metricsChan chan<- models.MessageLatencyMetric, wg *sync.WaitGroup) {
	defer wg.Done()

	// Setup the TLS configuration for the client
	caFile := filepath.Join(clientFolderPath, "ca.crt")
	certFile := filepath.Join(clientFolderPath, clientID, "client.chain.crt")
	keyFile := filepath.Join(clientFolderPath, clientID, "client.key")

	tlsConfig, err := utils.NewTLSConfig(caFile, certFile, keyFile)
	if err != nil {
		log.Printf("Consumer failed to load TLS config: %v", err)
		return
	}

	opts := mqtt.NewClientOptions().AddBroker(brokerAddress).SetClientID(clientID).SetTLSConfig(tlsConfig)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10*time.Second) || token.Error() != nil {
		log.Printf("Consumer failed to connect: %v", token.Error())
		return
	}
	defer client.Disconnect(250)

	//topic := "test/to/#" // Subscribe to all latency events
	latencies := make([]time.Duration, 0) // Holds latency for each received message
	var messageCount int64 = 0

	token = client.Subscribe(topic, byte(qos), func(client mqtt.Client, msg mqtt.Message) {
		timestampBytes := msg.Payload()[:8]
		sentTime := time.Unix(0, int64(binary.BigEndian.Uint64(timestampBytes)))
		latency := time.Since(sentTime)
		latencies = append(latencies, latency)
		messageCount++
	})
	if !token.WaitTimeout(10*time.Second) || token.Error() != nil {
		log.Printf("Consumer failed to subscribe: %v", token.Error())
		return
	}

	// Every second, print out the statistics for the previous second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Monitor messages for the specified test duration
	startTime := time.Now()

	for time.Since(startTime) < testDuration {
		select {
		case <-ticker.C:
			// At the end of each second, calculate the stats for the last second
			if len(latencies) > 0 {
				// Calculate stats for the current second
				totalLatency := time.Duration(0)
				minLatency := latencies[0]
				maxLatency := latencies[0]

				for _, latency := range latencies {
					totalLatency += latency
					if latency < minLatency {
						minLatency = latency
					}
					if latency > maxLatency {
						maxLatency = latency
					}
				}

				// Calculate average latency for this second
				averageLatency := totalLatency / time.Duration(len(latencies))

				// Print out the stats for this second
				log.Printf("[Real-Time] Messages/sec: %d, Avg Latency: %v, Min Latency: %v, Max Latency: %v",
					len(latencies), averageLatency, minLatency, maxLatency)

				// Reset latencies for the next second
				latencies = nil
			}
		}
	}

	// After the test duration ends, send the final metrics

	//	if len(latencies) > 0 {
	//		totalLatency := time.Duration(0)
	//		minLatency := latencies[0]
	//		maxLatency := latencies[0]
	//
	//		for _, latency := range latencies {
	//			totalLatency += latency
	//			if latency < minLatency {
	//				minLatency = latency
	//			}
	//			if latency > maxLatency {
	//				maxLatency = latency
	//			}
	//		}
	//
	//		metricsChan <- models.MessageLatencyMetric{
	//			TotalMessages:  messageCount,
	//			AverageLatency: float64(totalLatency.Milliseconds()) / float64(len(latencies)),
	//			MinLatency:     float64(minLatency.Milliseconds()),
	//			MaxLatency:     float64(maxLatency.Milliseconds()),
	//		}
	//	}
}

// RunThroughputTest orchestrates the latency test and generates the report.
func RunThroughputTest(config models.Config, qosLevel int) {
	var connectWg sync.WaitGroup // Synchronize client connections
	var messageWg sync.WaitGroup // Synchronize message sending

	messageChan := make(chan models.MessageMetric, config.MaxConnections)
	metricsChan := make(chan models.MessageLatencyMetric, 1)

	consumerClientID := fmt.Sprintf("LE40B8BBML1%06d", config.MaxConnections+1)
	topic := fmt.Sprintf("test/to/+")
	messageWg.Add(1)
	go func() {
		messageWg.Done()
		subscribeToTopic(consumerClientID, topic, config.BrokerAddress, config.ClientFolderPath, qosLevel, config.TestDuration.Duration, metricsChan, &messageWg)
	}()

	totalSuccessfulClients := 0
	totalFailedClients := 0
	totalConnectionTime := time.Duration(0)

	var connectMetrics []models.ConnectMetric

	// Shared end time for all clients
	testEndTime := time.Now().Add(config.TestDuration.Duration)

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
				go utils.SendMessages(client, id, fmt.Sprintf("test/to/%s", id), config.MessageSize, testEndTime, &config, messageChan, &messageWg)
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

		log.Printf("Batch %d: Success: %d, Fail: %d, Avg: %v, Min: %v, Max: %v",
			batch, batchSuccess, batchFail, avgConnTime, minConnTime, maxConnTime)
	}

	// Wait for all message goroutines to finish after the test duration
	messageWg.Wait()
	close(messageChan)

	// Generate message latency metrics
	var messageLatencyMetric models.MessageLatencyMetric
	for metric := range metricsChan {
		messageLatencyMetric = metric
	}
	fmt.Println(messageLatencyMetric.TotalMessages, messageLatencyMetric.AverageLatency, messageLatencyMetric.MinLatency, messageLatencyMetric.MaxLatency)
	close(metricsChan)

	// Collect message metrics
	totalMessagesSent := 0
	totalMessagesFailed := 0
	var messageMetrics []models.MessageMetric

	for metrics := range messageChan {
		totalMessagesSent += metrics.MessagesSent
		totalMessagesFailed += metrics.MessagesFailed
		messageMetrics = append(messageMetrics, metrics)
	}

	// Calculate throughput
	duration := config.TestDuration.Duration.Seconds()
	throughput := float64(totalMessagesSent) / duration

	// Prepare the report
	summary := models.ThroughputData{
		StartTime:             time.Time{},
		EndTime:               time.Time{},
		TotalMessagesSent:     totalMessagesSent,
		TotalMessagesFailed:   totalMessagesFailed,
		OverallThroughput:     throughput,
		MessageReceiveLatency: messageLatencyMetric,
	}

	// Write report to file
	reportFile, err := os.Create(fmt.Sprintf("%s/throughput_report.json", config.ReportDir))
	if err != nil {
		log.Fatalf("Failed to create report file: %v", err)
	}
	defer reportFile.Close()

	reportJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize report data: %v", err)
	}
	reportFile.Write(reportJSON)

	log.Printf("Event latency test completed. Report saved to %s", fmt.Sprintf("%s/throughput_report.json", config.ReportDir))
}
