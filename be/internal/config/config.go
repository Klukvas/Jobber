package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// PlanLimitsYAML mirrors model.PlanLimits with yaml tags for config loading.
type PlanLimitsYAML struct {
	MaxJobs           int `yaml:"max_jobs"`
	MaxResumes        int `yaml:"max_resumes"`
	MaxApplications   int `yaml:"max_applications"`
	MaxAIRequests     int `yaml:"max_ai_requests"`
	MaxJobParses      int `yaml:"max_job_parses"`
	MaxResumeBuilders int `yaml:"max_resume_builders"`
	MaxCoverLetters   int `yaml:"max_cover_letters"`
}

// Config holds all configuration for the application
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	JWT            JWTConfig
	Log            LogConfig
	S3             S3Config
	GoogleCalendar GoogleCalendarConfig
	Anthropic      AnthropicConfig
	Paddle         PaddleConfig
	Sentry         SentryConfig
	Resend         ResendConfig
	Features       FeaturesConfig
	Plans          map[string]PlanLimitsYAML
}

// FeaturesConfig holds feature flags that can be toggled per environment.
type FeaturesConfig struct {
	SentryEnabled   bool
	EmailEnabled    bool
	PaymentsEnabled bool
}

// SentryConfig holds Sentry error tracking configuration
type SentryConfig struct {
	DSN     string
	Release string
}

// ResendConfig holds Resend email service configuration
type ResendConfig struct {
	APIKey      string
	FromAddress string
}

// PaddleConfig holds Paddle payment configuration
type PaddleConfig struct {
	APIKey            string
	WebhookSecret     string
	Environment       string // sandbox or production
	ProPriceID        string
	EnterprisePriceID string
	ClientToken       string // frontend overlay checkout
}

// AnthropicConfig holds Anthropic API configuration
type AnthropicConfig struct {
	APIKey string
}

// GoogleCalendarConfig holds Google Calendar integration configuration
type GoogleCalendarConfig struct {
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	TokenEncryptionKey string // 64 hex chars = 32 bytes AES key
	FrontendURL        string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port           string
	Env            string
	AllowedOrigins string
	FrontendURL    string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	AccessSecret   string
	RefreshSecret  string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string
}

// S3Config holds S3 storage configuration
type S3Config struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			Env:            getEnv("SERVER_ENV", "development"),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
			FrontendURL:    getEnv("FRONTEND_URL", ""),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "jobber"),
			Password:        getEnv("DB_PASSWORD", "jobber"),
			DBName:          getEnv("DB_NAME", "jobber"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxConns:        getEnvAsInt("DB_MAX_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:   getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret:  getEnv("JWT_REFRESH_SECRET", ""),
			AccessExpiry:   getEnvAsDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry:  getEnvAsDuration("JWT_REFRESH_EXPIRY", 168*time.Hour),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		S3: S3Config{
			Endpoint:  getEnv("S3_ENDPOINT", ""),
			Bucket:    getEnv("S3_BUCKET", ""),
			Region:    getEnv("S3_REGION", "eu-central"),
			AccessKey: getEnv("S3_ACCESS_KEY", ""),
			SecretKey: getEnv("S3_SECRET_KEY", ""),
		},
		GoogleCalendar: GoogleCalendarConfig{
			ClientID:           getEnv("GOOGLE_CALENDAR_CLIENT_ID", ""),
			ClientSecret:       getEnv("GOOGLE_CALENDAR_CLIENT_SECRET", ""),
			RedirectURL:        getEnv("GOOGLE_CALENDAR_REDIRECT_URL", ""),
			TokenEncryptionKey: getEnv("GOOGLE_CALENDAR_TOKEN_ENCRYPTION_KEY", ""),
			FrontendURL:        getEnv("GOOGLE_CALENDAR_FRONTEND_URL", ""),
		},
		Anthropic: AnthropicConfig{
			APIKey: getEnv("ANTHROPIC_API_KEY", ""),
		},
		Paddle: PaddleConfig{
			APIKey:            getEnv("PADDLE_API_KEY", ""),
			WebhookSecret:     getEnv("PADDLE_WEBHOOK_SECRET", ""),
			Environment:       getEnv("PADDLE_ENVIRONMENT", "sandbox"),
			ProPriceID:        getEnv("PADDLE_PRO_PRICE_ID", ""),
			EnterprisePriceID: getEnv("PADDLE_ENTERPRISE_PRICE_ID", ""),
			ClientToken:       getEnv("PADDLE_CLIENT_TOKEN", ""),
		},
		Sentry: SentryConfig{
			DSN:     getEnv("SENTRY_DSN", ""),
			Release: getEnv("SENTRY_RELEASE", ""),
		},
		Resend: ResendConfig{
			APIKey:      getEnv("RESEND_API_KEY", ""),
			FromAddress: getEnv("RESEND_FROM_ADDRESS", ""),
		},
		Features: FeaturesConfig{
			SentryEnabled:   getEnvAsBool("FEATURE_SENTRY_ENABLED", true),
			EmailEnabled:    getEnvAsBool("FEATURE_EMAIL_ENABLED", true),
			PaymentsEnabled: getEnvAsBool("FEATURE_PAYMENTS_ENABLED", false),
		},
	}

	// Load plan limits from YAML (optional — falls back to hardcoded defaults)
	plansPath := getEnv("PLANS_CONFIG_PATH", "config/plans.yaml")
	cfg.Plans = loadPlansConfig(plansPath)

	// Validate required fields
	if cfg.JWT.AccessSecret == "" {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if cfg.JWT.RefreshSecret == "" {
		return nil, fmt.Errorf("JWT_REFRESH_SECRET is required")
	}

	// Production security guards
	if cfg.Server.Env == "production" {
		if cfg.Server.AllowedOrigins == "*" {
			return nil, fmt.Errorf("ALLOWED_ORIGINS must not be '*' in production")
		}
		if len(cfg.JWT.AccessSecret) < 32 {
			return nil, fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters in production")
		}
		if len(cfg.JWT.RefreshSecret) < 32 {
			return nil, fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters in production")
		}
	}

	return cfg, nil
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// RedisAddr returns the Redis address
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// loadPlansConfig reads plan limits from a YAML file.
// Returns nil if the file is missing (hardcoded defaults will be used).
func loadPlansConfig(path string) map[string]PlanLimitsYAML {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[config] plans config not found at %s, using hardcoded defaults", path)
		return nil
	}

	var plans map[string]PlanLimitsYAML
	if err := yaml.Unmarshal(data, &plans); err != nil {
		log.Printf("[config] failed to parse plans config %s: %v — using hardcoded defaults", path, err)
		return nil
	}

	log.Printf("[config] loaded plan limits from %s (%d plans)", path, len(plans))
	return plans
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
