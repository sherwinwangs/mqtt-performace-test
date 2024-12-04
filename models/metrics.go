package models

import "time"

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

type MessageLatencyMetric struct {
	TotalMessages  int64   `json:"total_messages"`
	AverageLatency float64 `json:"average_latency_ms"`
	MinLatency     float64 `json:"min_latency_ms"`
	MaxLatency     float64 `json:"max_latency_ms"`
}

type ThroughputData struct {
	StartTime             time.Time            `json:"start_time"`
	EndTime               time.Time            `json:"end_time"`
	TotalMessagesSent     int                  `json:"total_messages_sent"`
	TotalMessagesFailed   int                  `json:"total_messages_failed"`
	OverallThroughput     float64              `json:"overall_throughput"`
	MessageReceiveLatency MessageLatencyMetric `json:"message_receive_latency"`
}
type ConnectData struct {
	TotalConnectionsAttempted    int             `json:"total_connections_attempted"`
	TotalSuccessfulConnections   int             `json:"total_successful_connections"`
	TotalFailedConnections       int             `json:"total_failed_connections"`
	OverallAverageConnectionTime time.Duration   `json:"overall_average_connection_time"`
	AverageConnectionsPerSecond  float64         `json:"average_connections_per_second"`
	StartTime                    time.Time       `json:"start_time"`
	EndTime                      time.Time       `json:"end_time"`
	ConnectMetrics               []ConnectMetric `json:"connect_metrics"`
}

type SentData struct {
}
type SummaryData struct {
	Connect ConnectData `json:"connect"`
}
