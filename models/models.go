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
	ReportDir        string   `json:"report_dir"`
}
