package endpoints

import "time"

// Response structure for health checks
type HealthResponse struct {
	Status       string            `json:"status"`
	Timestamp    string            `json:"timestamp"`
	Uptime       string            `json:"uptime"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// Data represents the structure of the JSON response.
type Data struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// TimestampRange represents min and max timestamps for /query/range.
type TimestampRange struct {
	MinTimestamp time.Time `json:"minTimestamp"`
	MaxTimestamp time.Time `json:"maxTimestamp"`
}
