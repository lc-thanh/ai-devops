// Package ai provides the AI client interface and implementations.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ai-devops/internal/config"
	"github.com/ai-devops/internal/domain"
	"go.uber.org/zap"
)

// OpenAIClient implements the Client interface using OpenAI-compatible API.
type OpenAIClient struct {
	config       *config.AIConfig
	httpClient   *http.Client
	prompter     PromptBuilder
	validator    ResponseValidator
	logger       *zap.Logger
}

// OpenAI API request/response structures
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// NewOpenAIClient creates a new OpenAI-compatible AI client.
func NewOpenAIClient(cfg *config.AIConfig, prompter PromptBuilder, validator ResponseValidator, logger *zap.Logger) *OpenAIClient {
	return &OpenAIClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		prompter:  prompter,
		validator: validator,
		logger:    logger.Named("ai_client"),
	}
}

// Analyze sends a log to the AI service and returns a structured analysis.
func (c *OpenAIClient) Analyze(ctx context.Context, log string) (*domain.AnalysisResult, error) {
	startTime := time.Now()
	c.logger.Debug("starting AI analysis", zap.Int("log_length", len(log)))

	// Build the request
	reqBody := chatRequest{
		Model: c.config.Model,
		Messages: []chatMessage{
			{Role: "system", Content: c.prompter.BuildSystemPrompt()},
			{Role: "user", Content: c.prompter.BuildUserPrompt(log)},
		},
		MaxTokens:   c.config.MaxTokens,
		Temperature: 0.1, // Low temperature for deterministic output
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, domain.WrapError("marshal_request", err, false)
	}

	// Create HTTP request with context
	url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, domain.WrapError("create_request", err, false)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	// Execute request with retry logic
	var result *domain.AnalysisResult
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			c.logger.Debug("retrying AI request",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			select {
			case <-ctx.Done():
				return nil, domain.WrapError("context_cancelled", ctx.Err(), false)
			case <-time.After(backoff):
			}
		}

		result, lastErr = c.executeRequest(ctx, req)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if !domain.IsRetryable(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	c.logger.Debug("AI analysis completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.String("error_type", result.ErrorType),
	)

	return result, nil
}

// executeRequest performs a single HTTP request to the AI service.
func (c *OpenAIClient) executeRequest(ctx context.Context, req *http.Request) (*domain.AnalysisResult, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, domain.WrapError("ai_timeout", domain.ErrAITimeout, true)
		}
		return nil, domain.WrapError("http_request", err, true)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domain.WrapError("read_response", err, true)
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, domain.WrapError("rate_limit", domain.ErrRateLimited, true)
		}
		if resp.StatusCode >= 500 {
			return nil, domain.WrapError("ai_unavailable", domain.ErrAIUnavailable, true)
		}
		return nil, domain.WrapError("ai_error",
			fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(body)), false)
	}

	// Parse the response
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, domain.WrapError("parse_response", err, false)
	}

	if chatResp.Error != nil {
		return nil, domain.WrapError("ai_api_error",
			fmt.Errorf("%s: %s", chatResp.Error.Type, chatResp.Error.Message), false)
	}

	if len(chatResp.Choices) == 0 {
		return nil, domain.WrapError("empty_response", domain.ErrInvalidAIResponse, false)
	}

	// Extract and parse the JSON content from the response
	content := chatResp.Choices[0].Message.Content
	result, err := c.parseAnalysisResult(content)
	if err != nil {
		return nil, err
	}

	// Validate the result
	if err := c.validator.Validate(result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseAnalysisResult extracts the AnalysisResult from the AI response content.
func (c *OpenAIClient) parseAnalysisResult(content string) (*domain.AnalysisResult, error) {
	var result domain.AnalysisResult

	// Try to find JSON in the content (AI might include markdown code blocks)
	jsonContent := extractJSON(content)
	if jsonContent == "" {
		c.logger.Warn("could not extract JSON from AI response",
			zap.String("content_preview", truncate(content, 200)),
		)
		return nil, domain.WrapError("extract_json", domain.ErrInvalidAIResponse, false)
	}

	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		c.logger.Warn("failed to unmarshal AI response",
			zap.Error(err),
			zap.String("json_content", truncate(jsonContent, 200)),
		)
		return nil, domain.WrapError("unmarshal_result", domain.ErrInvalidAIResponse, false)
	}

	return &result, nil
}

// HealthCheck verifies the AI service is reachable.
func (c *OpenAIClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/models", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.WrapError("health_check", domain.ErrAIUnavailable, true)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.WrapError("health_check", domain.ErrAIUnavailable, true)
	}

	return nil
}

// Helper functions

// extractJSON attempts to extract JSON from content that might include markdown.
func extractJSON(content string) string {
	// Try to parse the entire content as JSON first
	if isValidJSON(content) {
		return content
	}

	// Look for JSON within markdown code blocks
	start := -1
	end := -1

	// Find opening brace
	for i, c := range content {
		if c == '{' {
			start = i
			break
		}
	}

	if start == -1 {
		return ""
	}

	// Find matching closing brace
	depth := 0
	for i := start; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end == -1 {
		return ""
	}

	extracted := content[start:end]
	if isValidJSON(extracted) {
		return extracted
	}

	return ""
}

func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
