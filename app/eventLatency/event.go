package eventLatency

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wukongcloud/mqtt-performace-test/app/connect"
	"github.com/wukongcloud/mqtt-performace-test/models"
	"github.com/wukongcloud/mqtt-performace-test/utils"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// subscribeToAllEvents listens for all event types using a wildcard subscription
func subscribeToAllEvents(clientID, brokerAddress, clientFolderPath string, testDuration time.Duration, metricsChan chan<- models.EventLatencyMetric) {
	// Setup the TLS configuration for the client
	caFile := filepath.Join(clientFolderPath, "ca.crt")
	certFile := filepath.Join(clientFolderPath, clientID, "client.chain.crt")
	keyFile := filepath.Join(clientFolderPath, clientID, "client.key")

	tlsConfig, err := utils.NewTLSConfig(caFile, certFile, keyFile)
	if err != nil {
		log.Printf("Consumer Client %s failed to load TLS config: %v", clientID, err)
		return
	}

	opts := mqtt.NewClientOptions().AddBroker(brokerAddress).SetClientID(clientID).SetTLSConfig(tlsConfig)
	client := mqtt.NewClient(opts)

	// Connect the consumer client
	if token := client.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		log.Printf("Consumer Client %s failed to connect: %v", clientID, token.Error())
		return
	}
	defer client.Disconnect(250)

	// Initialize the latency metrics for all events
	eventLatencyMetrics := make(map[string]*models.EventLatencyMetric)
	metricsMutex := &sync.Mutex{}

	// Subscribe to all event topics using a wildcard subscription
	topic := "$events/#"
	if token := client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		// Parse the event message
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &event); err != nil {
			log.Printf("Error unmarshalling event message: %v", err)
			return
		}

		// Extract the event type and timestamp
		eventType, ok := event["event"].(string)
		if !ok {
			log.Printf("Event message does not contain valid 'event' field: %v", event)
			return
		}

		// Extract timestamp from the event
		if timestamp, ok := event["timestamp"].(float64); ok {
			eventTimestamp := time.Unix(0, int64(timestamp*1e6))
			latency := time.Since(eventTimestamp)

			metricsMutex.Lock()
			defer metricsMutex.Unlock()
			if _, exists := eventLatencyMetrics[eventType]; !exists {
				eventLatencyMetrics[eventType] = &models.EventLatencyMetric{
					EventType:  eventType,
					MinLatency: float64(latency.Milliseconds()),
					MaxLatency: float64(latency.Milliseconds()),
				}
			}

			metrics := eventLatencyMetrics[eventType]
			metrics.EventCount++
			metrics.AverageLatency = ((metrics.AverageLatency * float64(metrics.EventCount-1)) + float64(latency.Milliseconds())) / float64(metrics.EventCount)

			if float64(latency.Milliseconds()) < metrics.MinLatency {
				metrics.MinLatency = float64(latency.Milliseconds())
			}
			if float64(latency.Milliseconds()) > metrics.MaxLatency {
				metrics.MaxLatency = float64(latency.Milliseconds())
			}
		} else {
			log.Printf("Event message does not contain valid 'timestamp' field: %v", event)
		}
	}); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		log.Printf("Consumer Client %s failed to subscribe to %s: %v", clientID, topic, token.Error())
		return
	}

	// Wait for messages for the given test duration
	done := make(chan struct{})
	go func() {
		time.Sleep(testDuration + time.Second*5)
		close(done)
	}()

	<-done // Wait for the test duration to complete

	// Ensure MQTT client processes any pending messages
	client.Disconnect(500)

	// Send the aggregated metrics for each event type
	metricsMutex.Lock()
	defer metricsMutex.Unlock()
	for _, metrics := range eventLatencyMetrics {
		metricsChan <- *metrics
		log.Printf("Event %s: Total: %d, Avg: %.2fms, Min: %.2fms, Max: %.2fms",
			metrics.EventType, metrics.EventCount, metrics.AverageLatency, metrics.MinLatency, metrics.MaxLatency)
	}
}

// EventLatencyTest performs latency measurement for all system events
func EventLatencyTest(config models.Config) {
	var wg sync.WaitGroup
	metricsChan := make(chan models.EventLatencyMetric, config.MaxConnections)

	// Start a consumer client to listen to all events
	consumerClientID := fmt.Sprintf("LE40B8BBML1%06d", config.MaxConnections+1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		subscribeToAllEvents(consumerClientID, config.BrokerAddress, config.ClientFolderPath, config.TestDuration.Duration, metricsChan)
	}()

	// 启动连接测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		connect.RunConnectTest(config)
	}()

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(metricsChan) // 确保在所有任务完成后关闭通道
	}()

	// Collect and process metrics for all event types
	eventLatencyMetrics := make([]models.EventLatencyMetric, 0)
	for metric := range metricsChan {
		eventLatencyMetrics = append(eventLatencyMetrics, metric)
	}

	// Write the aggregated metrics to the report
	report := map[string]interface{}{
		"event_latency_metrics": eventLatencyMetrics,
	}

	reportFile, err := os.Create(fmt.Sprintf("%s/event_report.json", config.ReportDir))
	if err != nil {
		log.Fatalf("Failed to create report file: %v", err)
	}
	defer reportFile.Close()

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize report data: %v", err)
	}
	reportFile.Write(reportJSON)

	log.Printf("Event latency test completed. Report saved to %s", fmt.Sprintf("%s/event_report.json", config.ReportDir))
}
