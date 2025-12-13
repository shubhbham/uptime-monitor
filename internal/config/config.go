package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Monitor  MonitorConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Environment  string
}

type DatabaseConfig struct {
	URL      string
	SSLMode  string
	MaxConns int
	MinConns int
}

type MonitorConfig struct {
	WorkerPoolSize     int
	MaxRetries         int
	DefaultTimeout     time.Duration
	StatsUpdatePeriod  time.Duration
}

func LoadConfig() (*Config, error) {
	port, err := getEnvAsInt("PORT", 5000)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	readTimeout, err := getEnvAsInt("READ_TIMEOUT", 10)
	if err != nil {
		return nil, fmt.Errorf("invalid READ_TIMEOUT: %w", err)
	}

	writeTimeout, err := getEnvAsInt("WRITE_TIMEOUT", 10)
	if err != nil {
		return nil, fmt.Errorf("invalid WRITE_TIMEOUT: %w", err)
	}

	dbMaxConns, err := getEnvAsInt("DB_MAX_CONNS", 25)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_CONNS: %w", err)
	}

	dbMinConns, err := getEnvAsInt("DB_MIN_CONNS", 5)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MIN_CONNS: %w", err)
	}

	workerPoolSize, err := getEnvAsInt("WORKER_POOL_SIZE", 10)
	if err != nil {
		return nil, fmt.Errorf("invalid WORKER_POOL_SIZE: %w", err)
	}

	maxRetries, err := getEnvAsInt("MAX_RETRIES", 3)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_RETRIES: %w", err)
	}

	defaultTimeout, err := getEnvAsInt("DEFAULT_TIMEOUT", 10)
	if err != nil {
		return nil, fmt.Errorf("invalid DEFAULT_TIMEOUT: %w", err)
	}

	statsUpdatePeriod, err := getEnvAsInt("STATS_UPDATE_PERIOD", 60)
	if err != nil {
		return nil, fmt.Errorf("invalid STATS_UPDATE_PERIOD: %w", err)
	}

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &Config{
		Server: ServerConfig{
			Port:         port,
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
			Environment:  getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			URL:      databaseURL,
			SSLMode:  getEnv("DB_SSL_MODE", "require"),
			MaxConns: dbMaxConns,
			MinConns: dbMinConns,
		},
		Monitor: MonitorConfig{
			WorkerPoolSize:    workerPoolSize,
			MaxRetries:        maxRetries,
			DefaultTimeout:    time.Duration(defaultTimeout) * time.Second,
			StatsUpdatePeriod: time.Duration(statsUpdatePeriod) * time.Second,
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) (int, error) {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(valueStr)
}