// Package config handles application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ai-devops/internal/domain"
)

// Config holds all application configuration.
type Config struct {
	// Server configuration
	Server ServerConfig

	// AI service configuration
	AI AIConfig

	// Log processing configuration
	Processing ProcessingConfig
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	// Port is the HTTP port to listen on.
	Port string

	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response.
	WriteTimeout time.Duration
}

// AIProvider represents the AI provider to use.
type AIProvider string

const (
	// AIProviderOpenAI uses OpenAI-compatible API.
	AIProviderOpenAI AIProvider = "openai"

	// AIProviderGemini uses Google Gemini API.
	AIProviderGemini AIProvider = "gemini"
)

// AIConfig contains AI service settings.
type AIConfig struct {
	// Provider specifies which AI provider to use (openai, gemini).
	Provider AIProvider

	// APIKey is the authentication key for the AI provider.
	APIKey string

	// BaseURL is the base URL for the AI API (optional, provider-specific defaults).
	BaseURL string

	// Model is the AI model to use.
	Model string

	// Timeout is the maximum time to wait for AI responses.
	Timeout time.Duration

	// MaxTokens is the maximum tokens for AI response.
	MaxTokens int

	// MaxRetries is the number of retries on transient failures.
	MaxRetries int

	// MockMode enables mock responses for testing without API calls.
	MockMode bool
}

// ProcessingConfig contains log processing settings.
type ProcessingConfig struct {
	// MaxLogSize is the maximum allowed log size in bytes.
	MaxLogSize int

	// EnableRules enables rule-based pre-classification.
	EnableRules bool

	// RuleConfidenceThreshold is the minimum confidence to use rule results.
	RuleConfidenceThreshold float64
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Determine AI provider
	provider := AIProvider(getEnvOrDefault("AI_PROVIDER", "openai"))

	// Set provider-specific defaults
	var defaultBaseURL, defaultModel string
	switch provider {
	case AIProviderGemini:
		defaultBaseURL = "https://generativelanguage.googleapis.com"
		defaultModel = "gemini-2.0-flash"
	default:
		provider = AIProviderOpenAI
		defaultBaseURL = "https://api.openai.com/v1"
		defaultModel = "gpt-4o-mini"
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvOrDefault("PORT", "8080"),
			ReadTimeout:  getDurationOrDefault("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationOrDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		AI: AIConfig{
			Provider:   provider,
			APIKey:     os.Getenv("AI_API_KEY"),
			BaseURL:    getEnvOrDefault("AI_BASE_URL", defaultBaseURL),
			Model:      getEnvOrDefault("AI_MODEL", defaultModel),
			Timeout:    getDurationOrDefault("AI_TIMEOUT", 30*time.Second),
			MaxTokens:  getIntOrDefault("AI_MAX_TOKENS", 1024),
			MaxRetries: getIntOrDefault("AI_MAX_RETRIES", 2),
			MockMode:   getBoolOrDefault("AI_MOCK_MODE", false),
		},
		Processing: ProcessingConfig{
			MaxLogSize:              getIntOrDefault("MAX_LOG_SIZE", 50000), // ~50KB
			EnableRules:             getBoolOrDefault("ENABLE_RULES", true),
			RuleConfidenceThreshold: getFloatOrDefault("RULE_CONFIDENCE_THRESHOLD", 0.8),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// AI API key is required unless in mock mode
	if !c.AI.MockMode && c.AI.APIKey == "" {
		return fmt.Errorf("%w: AI_API_KEY is required when not in mock mode", domain.ErrInvalidConfig)
	}

	if c.AI.Timeout < time.Second {
		return fmt.Errorf("%w: AI_TIMEOUT must be at least 1 second", domain.ErrInvalidConfig)
	}

	if c.AI.MaxTokens < 100 {
		return fmt.Errorf("%w: AI_MAX_TOKENS must be at least 100", domain.ErrInvalidConfig)
	}

	if c.Processing.MaxLogSize < 1000 {
		return fmt.Errorf("%w: MAX_LOG_SIZE must be at least 1000 bytes", domain.ErrInvalidConfig)
	}

	if c.Processing.RuleConfidenceThreshold < 0 || c.Processing.RuleConfidenceThreshold > 1 {
		return fmt.Errorf("%w: RULE_CONFIDENCE_THRESHOLD must be between 0 and 1", domain.ErrInvalidConfig)
	}

	return nil
}

// Helper functions for reading environment variables

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getBoolOrDefault(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

func getFloatOrDefault(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		// Try parsing as seconds first (e.g., "15")
		if secs, err := strconv.Atoi(val); err == nil {
			return time.Duration(secs) * time.Second
		}
		// Try parsing as duration string (e.g., "15s", "1m")
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
