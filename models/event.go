package models

// EventLatencyMetric stores aggregated metrics for each event type
type EventLatencyMetric struct {
	EventType      string  `json:"event_type"`
	EventCount     uint64  `json:"event_count"`
	AverageLatency float64 `json:"average_latency_ms"`
	MinLatency     float64 `json:"min_latency_ms"`
	MaxLatency     float64 `json:"max_latency_ms"`
}
