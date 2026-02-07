// Package ai provides the AI client interface and implementations.
package ai

import (
	"context"

	"github.com/ai-devops/internal/domain"
	"go.uber.org/zap"
)

// MockClient implements the Client interface for testing.
type MockClient struct {
	logger *zap.Logger
}

// NewMockClient creates a new mock AI client for testing.
func NewMockClient(logger *zap.Logger) *MockClient {
	return &MockClient{
		logger: logger.Named("mock_ai_client"),
	}
}

// Analyze returns a mock analysis result.
func (c *MockClient) Analyze(ctx context.Context, log string) (*domain.AnalysisResult, error) {
	c.logger.Debug("mock AI analysis", zap.Int("log_length", len(log)))

	// Return a generic mock response
	return &domain.AnalysisResult{
		ErrorType: "mock_error",
		Severity:  domain.SeverityMedium,
		RootCause: "This is a mock response. Enable real AI by setting AI_MOCK_MODE=false",
		SuggestedActions: []string{
			"Configure AI_API_KEY environment variable",
			"Set AI_MOCK_MODE=false to enable real AI analysis",
		},
		PreventionTips: []string{
			"Use real AI for production analysis",
		},
	}, nil
}

// HealthCheck always returns success for mock client.
func (c *MockClient) HealthCheck(ctx context.Context) error {
	return nil
}
