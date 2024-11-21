package models

import (
	"fmt"
	"time"
)

type Duration struct {
	time.Duration
}

// UnmarshalJSON parses a JSON string like "2s" into a `time.Duration`.
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Remove quotes from the JSON string
	str := string(b)
	if len(str) > 1 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// Parse the duration
	duration, err := time.ParseDuration(str)
	if err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}

	d.Duration = duration
	return nil
}

type Config struct {
	BrokerAddress    string   `json:"broker_address"`
	ClientFolderPath string   `json:"client_folder_path"` // Path to folder containing client certificates
	IncrementStep    int      `json:"increment_step"`     // Number of clients to add per iteration
	MaxConnections   int      `json:"max_connections"`    // Maximum number of connections to test
	ConnectTimeout   Duration `json:"connect_timeout"`    // timeout seconds
	MessageSize      int      `json:"message_size"`
	MessageFrequency Duration `json:"message_frequency"`
	TestDuration     Duration `json:"test_duration"`
	ReportFile       string   `json:"report_file"`
}

type ConnectionData struct {
	ClientID        string        `json:"client_id"`
	ConnectionTime  time.Duration `json:"connection_time"`
	ConnectionError bool          `json:"connection_error"`
}

type MessageMetric struct {
	ClientID       string `json:"client_id"`
	MessagesSent   int    `json:"messages_sent"`
	MessagesFailed int    `json:"messages_failed"`
}

type ConnectMetric struct {
	BatchNumber           int           `json:"batch_number"`
	SuccessfulConnections int           `json:"successful_connections"`
	FailedConnections     int           `json:"failed_connections"`
	AverageConnectionTime time.Duration `json:"average_connection_time"`
	MinConnectionTime     time.Duration `json:"min_connection_time"`
	MaxConnectionTime     time.Duration `json:"max_connection_time"`
}

type SummaryData struct {
	TotalConnectionsAttempted    int             `json:"total_connections_attempted"`
	TotalSuccessfulConnections   int             `json:"total_successful_connections"`
	TotalFailedConnections       int             `json:"total_failed_connections"`
	OverallAverageConnectionTime time.Duration   `json:"overall_average_connection_time"`
	AverageConnectionsPerSecond  float64         `json:"average_connections_per_second"`
	StartTime                    time.Time       `json:"start_time"`
	EndTime                      time.Time       `json:"end_time"`
	ConnectMetrics               []ConnectMetric `json:"connect_metrics"`
	TotalMessagesSent            int             `json:"total_messages_sent"`
	TotalMessagesFailed          int             `json:"total_messages_failed"`
	OverallThroughput            float64         `json:"overall_throughput"`
	//MessageMetrics               []MessageMetric `json:"client_metrics"`
}
