package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"smarthome/models"
)

// DeviceService handles sensor CRUD via the external device-api
type DeviceService struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewDeviceService creates a new DeviceService
func NewDeviceService(baseURL string) *DeviceService {
	return &DeviceService{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetSensors fetches all sensors from the device-api
func (s *DeviceService) GetSensors() ([]models.Sensor, error) {
	resp, err := s.HTTPClient.Get(s.BaseURL + "/api/v1/sensors")
	if err != nil {
		return nil, fmt.Errorf("error fetching sensors: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sensors []models.Sensor
	if err := json.NewDecoder(resp.Body).Decode(&sensors); err != nil {
		return nil, fmt.Errorf("error decoding sensors: %w", err)
	}
	return sensors, nil
}

// GetSensorByID fetches a single sensor by ID from the device-api
func (s *DeviceService) GetSensorByID(id int) (models.Sensor, error) {
	resp, err := s.HTTPClient.Get(fmt.Sprintf("%s/api/v1/sensors/%d", s.BaseURL, id))
	if err != nil {
		return models.Sensor{}, fmt.Errorf("error fetching sensor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return models.Sensor{}, fmt.Errorf("sensor not found")
	}
	if resp.StatusCode != http.StatusOK {
		return models.Sensor{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sensor models.Sensor
	if err := json.NewDecoder(resp.Body).Decode(&sensor); err != nil {
		return models.Sensor{}, fmt.Errorf("error decoding sensor: %w", err)
	}
	return sensor, nil
}

// CreateSensor creates a new sensor via the device-api
func (s *DeviceService) CreateSensor(sc models.SensorCreate) (models.Sensor, error) {
	body, err := json.Marshal(sc)
	if err != nil {
		return models.Sensor{}, err
	}

	resp, err := s.HTTPClient.Post(s.BaseURL+"/api/v1/sensors", "application/json", bytes.NewReader(body))
	if err != nil {
		return models.Sensor{}, fmt.Errorf("error creating sensor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return models.Sensor{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sensor models.Sensor
	if err := json.NewDecoder(resp.Body).Decode(&sensor); err != nil {
		return models.Sensor{}, fmt.Errorf("error decoding sensor: %w", err)
	}
	return sensor, nil
}

// UpdateSensor updates an existing sensor via the device-api
func (s *DeviceService) UpdateSensor(id int, su models.SensorUpdate) (models.Sensor, error) {
	body, err := json.Marshal(su)
	if err != nil {
		return models.Sensor{}, err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v1/sensors/%d", s.BaseURL, id), bytes.NewReader(body))
	if err != nil {
		return models.Sensor{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return models.Sensor{}, fmt.Errorf("error updating sensor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Sensor{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sensor models.Sensor
	if err := json.NewDecoder(resp.Body).Decode(&sensor); err != nil {
		return models.Sensor{}, fmt.Errorf("error decoding sensor: %w", err)
	}
	return sensor, nil
}

// DeleteSensor deletes a sensor via the device-api
func (s *DeviceService) DeleteSensor(id int) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/sensors/%d", s.BaseURL, id), nil)
	if err != nil {
		return err
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error deleting sensor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

// UpdateSensorValue updates the value and status of a sensor via the device-api
func (s *DeviceService) UpdateSensorValue(id int, value float64, status string) error {
	body, err := json.Marshal(map[string]interface{}{"value": value, "status": status})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/v1/sensors/%d/value", s.BaseURL, id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error updating sensor value: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
