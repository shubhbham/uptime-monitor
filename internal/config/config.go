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
	Auth     AuthConfig
	Email    EmailConfig
}

type EmailConfig struct {
	BrevoAPIKey              string
	BrevoSandboxMode         bool
	BrevoAPIBaseURL          string
	AlertFromEmail           string
	AlertFromName            string
	AlertTagIncident         string
	AlertTagService          string
	AlertOnIncidentOpen      bool
	AlertOnIncidentResolve   bool
	AlertsEnabled            bool
	AlertMaxRetries          int
	AlertRetryBackoffSeconds int
	AlertCooldownMinutes     int
	AlertReplyToEmail        string
	AlertReplyToName         string
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

type AuthConfig struct {
	// Clerk configuration
	ClerkSecretKey      string
	ClerkJWKSURL        string
	ClerkPublishableKey string

	// API Key configuration
	APIKeyPrefix        string
	APIKeyLength        int

	// JWT configuration
	// NOTE: JWT_EXPIRATION_HOURS is reserved for future use when implementing
	// custom JWT token generation. Currently, the system uses Clerk for JWT
	// authentication (external) and API Keys for programmatic access.
	JWTExpirationHours  int
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

	// Auth configuration
	clerkSecretKey := getEnv("CLERK_SECRET_KEY", "")
	clerkJWKSURL := getEnv("CLERK_JWKS_URL", "https://intense-moth-17.clerk.accounts.dev/.well-known/jwks.json")
	clerkPublishableKey := getEnv("CLERK_PUBLISHABLE_KEY", "")

	apiKeyPrefix := getEnv("API_KEY_PREFIX", "sk_live_")
	apiKeyLength, _ := getEnvAsInt("API_KEY_LENGTH", 32)
	// Reserved for future use - custom JWT token generation
	jwtExpirationHours, _ := getEnvAsInt("JWT_EXPIRATION_HOURS", 24)

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
		Auth: AuthConfig{
			ClerkSecretKey:      clerkSecretKey,
			ClerkJWKSURL:        clerkJWKSURL,
			ClerkPublishableKey: clerkPublishableKey,
			APIKeyPrefix:        apiKeyPrefix,
			APIKeyLength:        apiKeyLength,
			JWTExpirationHours:  jwtExpirationHours,
		},
		Email: EmailConfig{
			BrevoAPIKey:            getEnv("BREVO_API_KEY", ""),
			BrevoSandboxMode:       getEnv("BREVO_SANDBOX_MODE", "false") == "true",
			BrevoAPIBaseURL:        getEnv("BREVO_API_BASE_URL", "https://api.brevo.com"),
			AlertFromEmail:         getEnv("ALERT_FROM_EMAIL", ""),
			AlertFromName:          getEnv("ALERT_FROM_NAME", "Uptime Monitor"),
			AlertTagIncident:       getEnv("ALERT_TAG_INCIDENT", "incident"),
			AlertTagService:        getEnv("ALERT_TAG_SERVICE", "uptime-monitor"),
			AlertOnIncidentOpen:    getEnv("ALERT_ON_INCIDENT_OPEN", "true") == "true",
			AlertOnIncidentResolve: getEnv("ALERT_ON_INCIDENT_RESOLVE", "true") == "true",
			AlertsEnabled:          getEnv("ALERTS_ENABLED", "true") == "true",
			AlertMaxRetries:        getEnvInt("ALERT_MAX_RETRIES", 3),
			AlertRetryBackoffSeconds: getEnvInt("ALERT_RETRY_BACKOFF_SECONDS", 30),
			AlertCooldownMinutes:   getEnvInt("ALERT_COOLDOWN_MINUTES", 30),
			AlertReplyToEmail:      getEnv("ALERT_REPLY_TO_EMAIL", "no-reply@uptime.local"),
			AlertReplyToName:       getEnv("ALERT_REPLY_TO_NAME", "No Reply"),
		},
	}, nil
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return val
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