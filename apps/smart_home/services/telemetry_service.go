package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelemetryRecord represents a single stored sensor reading
type TelemetryRecord struct {
	ID         int       `json:"id"`
	SensorID   int       `json:"sensor_id"`
	Value      float64   `json:"value"`
	Status     string    `json:"status"`
	RecordedAt time.Time `json:"recorded_at"`
}

// TelemetryService fetches historical telemetry from the telemetry-api
type TelemetryService struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewTelemetryService creates a new TelemetryService
func NewTelemetryService(baseURL string) *TelemetryService {
	return &TelemetryService{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetTelemetry fetches telemetry records for a sensor within a time window
func (s *TelemetryService) GetTelemetry(sensorID int, from, to time.Time) ([]TelemetryRecord, error) {
	url := fmt.Sprintf("%s/api/v1/telemetry/%d?from=%s&to=%s",
		s.BaseURL, sensorID,
		from.UTC().Format(time.RFC3339),
		to.UTC().Format(time.RFC3339))

	resp, err := s.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching telemetry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var records []TelemetryRecord
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("error decoding telemetry: %w", err)
	}
	return records, nil
}
