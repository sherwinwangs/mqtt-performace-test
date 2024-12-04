package utils

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/wukongcloud/mqtt-performace-test/models"
	"io/ioutil"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func NewTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read CA cert file: %v", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("unable to append CA cert to pool")
	}

	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load client cert and key: %v", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS12,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientCert},
	}

	return tlsConfig, nil
}

func GenerateMessage(size int) []byte {
	msg := make([]byte, size)
	rand.Read(msg)
	return []byte(base64.StdEncoding.EncodeToString(msg)[:size])
}

// GenerateMessageWithTimestamp creates a payload with a timestamp prepended.
func GenerateMessageWithTimestamp(size int) []byte {
	payload := make([]byte, size)
	_, _ = rand.Read(payload)

	timestamp := time.Now().UnixNano() // Ensure the timestamp is updated with each call
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))

	return append(timestampBytes, payload...)
}

// SendMessages sends messages for a connected client until the test end time
func SendMessages(client mqtt.Client, clientID string, topic string, messageSize int, testEndTime time.Time, config *models.Config, metricsChan chan<- models.MessageMetric, wg *sync.WaitGroup) {
	defer wg.Done()

	messagesSent := 0
	messagesFailed := 0
	ticker := time.NewTicker(config.MessageFrequency.Duration)

	for range ticker.C {
		if time.Now().After(testEndTime) {
			break
		}

		message := GenerateMessageWithTimestamp(messagesSent)
		token := client.Publish(topic, 1, false, message)
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			messagesFailed++
		} else {
			messagesSent++
		}
	}

	metricsChan <- models.MessageMetric{
		ClientID:       clientID,
		MessagesSent:   messagesSent,
		MessagesFailed: messagesFailed,
	}
	// Debug
	// log.Printf("Client %s finished sending messages. Sent: %d, Failed: %d", clientID, messagesSent, messagesFailed)
}
