// Package ai provides the AI client interface and implementations.
package ai

import (
	"context"

	"github.com/ai-devops/internal/domain"
)

// Client defines the interface for AI service interactions.
// This interface allows for easy mocking and swapping of AI providers.
type Client interface {
	// Analyze sends a log to the AI service and returns a structured analysis.
	// The context should carry timeout and cancellation signals.
	Analyze(ctx context.Context, log string) (*domain.AnalysisResult, error)

	// HealthCheck verifies the AI service is reachable.
	HealthCheck(ctx context.Context) error
}

// PromptBuilder defines the interface for constructing AI prompts.
type PromptBuilder interface {
	// BuildSystemPrompt returns the system prompt that defines the AI's role.
	BuildSystemPrompt() string

	// BuildUserPrompt constructs the user prompt with the log content.
	BuildUserPrompt(log string) string
}

// ResponseValidator defines the interface for validating AI responses.
type ResponseValidator interface {
	// Validate checks if the AI response conforms to the expected schema.
	Validate(result *domain.AnalysisResult) error
}
