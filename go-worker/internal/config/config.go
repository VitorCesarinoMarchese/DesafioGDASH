package config

import (
	"os"
	"strconv"
)

// Config holds all configuration values for the worker
type Config struct {
	RabbitMQ RabbitMQConfig
	API      APIConfig
	Worker   WorkerConfig
}

// RabbitMQConfig holds RabbitMQ connection settings
type RabbitMQConfig struct {
	URL           string
	Host          string
	User          string
	Pass          string
	Queue         string
	PrefetchCount int
}

// APIConfig holds API client settings
type APIConfig struct {
	BaseURL       string
	Endpoint      string
	TimeoutMs     int
	RetryAttempts int
	RetryDelayMs  int
}

// WorkerConfig holds worker-specific settings
type WorkerConfig struct {
	// Add worker-specific config here if needed
}

// Load creates a new Config with values from environment variables
func Load() *Config {
	rabbitURL := getenvDefault("RABBITMQ_URL", "")
	rabbitHost := getenvDefault("RABBITMQ_HOST", "rabbitmq")
	rabbitUser := getenvDefault("RABBITMQ_DEFAULT_USER", "guest")
	rabbitPass := getenvDefault("RABBITMQ_DEFAULT_PASS", "guest")

	if rabbitURL == "" {
		rabbitURL = "amqp://" + rabbitUser + ":" + rabbitPass + "@" + rabbitHost + ":5672/"
	}

	return &Config{
		RabbitMQ: RabbitMQConfig{
			URL:           rabbitURL,
			Host:          rabbitHost,
			User:          rabbitUser,
			Pass:          rabbitPass,
			Queue:         getenvDefault("RABBITMQ_QUEUE", "weather"),
			PrefetchCount: getenvIntDefault("RABBITMQ_PREFETCH", 1),
		},
		API: APIConfig{
			BaseURL:       getenvDefault("API_URL", "http://localhost:3000"),
			Endpoint:      getenvDefault("NESTJS_ENDPOINT", "/api/weather/logs"),
			TimeoutMs:     getenvIntDefault("HTTP_TIMEOUT_MS", 10000),
			RetryAttempts: getenvIntDefault("RETRY_ATTEMPTS", 3),
			RetryDelayMs:  getenvIntDefault("RETRY_DELAY_MS", 2000),
		},
		Worker: WorkerConfig{},
	}
}

func getenvDefault(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func getenvIntDefault(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if ival, err := strconv.Atoi(v); err == nil {
			return ival
		}
	}
	return def
}