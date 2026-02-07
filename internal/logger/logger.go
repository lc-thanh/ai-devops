// Package logger provides structured logging setup.
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new structured logger.
func New(development bool) (*zap.Logger, error) {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Check for LOG_LEVEL environment variable
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		var zapLevel zapcore.Level
		if err := zapLevel.UnmarshalText([]byte(level)); err == nil {
			config.Level = zap.NewAtomicLevelAt(zapLevel)
		}
	}

	return config.Build()
}

// NewNop creates a no-op logger for testing.
func NewNop() *zap.Logger {
	return zap.NewNop()
}
