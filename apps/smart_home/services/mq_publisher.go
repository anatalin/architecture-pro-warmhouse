package services

import (
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const sensorTelemetryQueue = "sensor_telemetry"

type sensorTelemetryMsg struct {
	SensorID int     `json:"sensor_id"`
	Value    float64 `json:"value"`
	Status   string  `json:"status"`
}

// MQPublisher publishes sensor telemetry messages to RabbitMQ
type MQPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewMQPublisher connects to RabbitMQ and declares the telemetry queue
func NewMQPublisher(url string) (*MQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if _, err = ch.QueueDeclare(sensorTelemetryQueue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &MQPublisher{conn: conn, ch: ch}, nil
}

// Publish sends a sensor value update event to the telemetry queue
func (p *MQPublisher) Publish(sensorID int, value float64, status string) error {
	body, err := json.Marshal(sensorTelemetryMsg{SensorID: sensorID, Value: value, Status: status})
	if err != nil {
		return err
	}
	return p.ch.Publish("", sensorTelemetryQueue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

// Close releases RabbitMQ resources
func (p *MQPublisher) Close() {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
