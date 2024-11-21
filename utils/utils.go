package utils

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/wukongcloud/mqtt-performace-test/models"
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

// SendMessages sends messages for a connected client until the test end time
func SendMessages(client mqtt.Client, clientID string, topic string, message []byte, testEndTime time.Time, config *models.Config, metricsChan chan<- models.MessageMetric, wg *sync.WaitGroup) {
	defer wg.Done()

	messagesSent := 0
	messagesFailed := 0
	ticker := time.NewTicker(config.MessageFrequency.Duration)

	for range ticker.C {
		if time.Now().After(testEndTime) {
			ticker.Stop()
			break
		}

		token := client.Publish(topic, 1, false, message)
		if token.WaitTimeout(config.ConnectTimeout.Duration) && token.Error() != nil {
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
	log.Printf("Client %s finished sending messages. Sent: %d, Failed: %d", clientID, messagesSent, messagesFailed)
}
