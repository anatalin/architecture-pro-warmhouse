package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smarthome/handlers"
	"smarthome/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize temperature service
	temperatureAPIURL := getEnv("TEMPERATURE_API_URL", "http://temperature-api:8080")
	temperatureService := services.NewTemperatureService(temperatureAPIURL)
	log.Printf("Temperature service initialized with API URL: %s\n", temperatureAPIURL)

	// Initialize device service
	deviceAPIURL := getEnv("DEVICE_API_URL", "http://device-api:8080")
	deviceService := services.NewDeviceService(deviceAPIURL)
	log.Printf("Device service initialized with API URL: %s\n", deviceAPIURL)

	// Initialize RabbitMQ publisher
	mqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq/")
	mqPublisher, err := services.NewMQPublisher(mqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v\n", err)
	}
	defer mqPublisher.Close()
	log.Printf("RabbitMQ publisher initialized with URL: %s\n", mqURL)

	// Initialize telemetry service
	telemetryAPIURL := getEnv("TELEMETRY_API_URL", "http://telemetry-api:8080")
	telemetryService := services.NewTelemetryService(telemetryAPIURL)
	log.Printf("Telemetry service initialized with API URL: %s\n", telemetryAPIURL)

	// Initialize router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API routes
	apiRoutes := router.Group("/api/v1")

	// Register sensor routes
	sensorHandler := handlers.NewSensorHandler(deviceService, temperatureService, mqPublisher, telemetryService)
	sensorHandler.RegisterRoutes(apiRoutes)

	// Start server
	srv := &http.Server{
		Addr:    getEnv("PORT", ":8080"),
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	log.Println("Server exited properly")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
