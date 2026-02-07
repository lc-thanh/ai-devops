// Package ai provides the AI client interface and implementations.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-devops/internal/config"
	"github.com/ai-devops/internal/domain"
	"go.uber.org/zap"
)

// GeminiClient implements the Client interface using Google's Gemini API.
type GeminiClient struct {
	config     *config.AIConfig
	httpClient *http.Client
	prompter   PromptBuilder
	validator  ResponseValidator
	logger     *zap.Logger
}

// Gemini API request/response structures

// geminiRequest represents the request body for Gemini API.
type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig  `json:"generationConfig"`
	SafetySettings    []geminiSafetySetting   `json:"safetySettings,omitempty"`
}

// geminiSystemInstruction represents the system instruction for Gemini.
type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

// geminiContent represents a content block in Gemini API.
type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

// geminiPart represents a part of content (text, image, etc).
// Gemini thinking models may return additional fields.
type geminiPart struct {
	Text   string `json:"text,omitempty"`
	Thought string `json:"thought,omitempty"` // For thinking/reasoning models
}

// geminiGenerationConfig contains generation parameters.
type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

// geminiSafetySetting represents a safety setting for content filtering.
type geminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// geminiResponse represents the response from Gemini API.
type geminiResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
	Error          *geminiError          `json:"error,omitempty"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
}

// geminiUsageMetadata contains token usage info.
type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// geminiCandidate represents a response candidate.
type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	Index         int           `json:"index,omitempty"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
	// For thinking/reasoning models
	GroundingMetadata json.RawMessage `json:"groundingMetadata,omitempty"`
}

// geminiPromptFeedback contains feedback about the prompt.
type geminiPromptFeedback struct {
	BlockReason   string `json:"blockReason,omitempty"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

// geminiError represents an error response from Gemini API.
type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NewGeminiClient creates a new Gemini AI client.
func NewGeminiClient(cfg *config.AIConfig, prompter PromptBuilder, validator ResponseValidator, logger *zap.Logger) *GeminiClient {
	return &GeminiClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		prompter:  prompter,
		validator: validator,
		logger:    logger.Named("gemini_client"),
	}
}

// Analyze sends a log to the Gemini API and returns a structured analysis.
func (c *GeminiClient) Analyze(ctx context.Context, log string) (*domain.AnalysisResult, error) {
	startTime := time.Now()
	c.logger.Debug("starting Gemini analysis", zap.Int("log_length", len(log)))

	// Build the user prompt with system context embedded
	// Combine system prompt and user prompt for better compatibility
	systemPrompt := c.prompter.BuildSystemPrompt()
	userPrompt := c.prompter.BuildUserPrompt(log)
	combinedPrompt := fmt.Sprintf("%s\n\n---\n\n%s", systemPrompt, userPrompt)

	// Calculate max tokens - thinking models (2.5+) need more tokens
	// since thinking tokens count against the output limit
	maxTokens := c.config.MaxTokens
	if isThinkingModel(c.config.Model) {
		// Thinking models need ~4x more tokens to account for reasoning
		maxTokens = c.config.MaxTokens * 4
		if maxTokens < 4096 {
			maxTokens = 4096
		}
		c.logger.Debug("using increased token limit for thinking model",
			zap.String("model", c.config.Model),
			zap.Int("max_tokens", maxTokens),
		)
	}

	// Build the request using the contents array (more compatible approach)
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: combinedPrompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.1, // Low temperature for deterministic output
			MaxOutputTokens: maxTokens,
			TopP:            0.95,
			TopK:            40,
		},
		SafetySettings: []geminiSafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, domain.WrapError("marshal_request", err, false)
	}

	// Build the URL with API key as query parameter
	url := c.buildURL()

	// Execute request with retry logic
	var result *domain.AnalysisResult
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			c.logger.Debug("retrying Gemini request",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			select {
			case <-ctx.Done():
				return nil, domain.WrapError("context_cancelled", ctx.Err(), false)
			case <-time.After(backoff):
			}
		}

		result, lastErr = c.executeRequest(ctx, url, jsonBody)
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

	c.logger.Debug("Gemini analysis completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.String("error_type", result.ErrorType),
	)

	return result, nil
}

// buildURL constructs the Gemini API URL.
func (c *GeminiClient) buildURL() string {
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")

	// Support both full URL and just the base
	if strings.Contains(baseURL, "/v1") || strings.Contains(baseURL, "/v1beta") {
		// URL already contains version, append model and action
		return fmt.Sprintf("%s/models/%s:generateContent?key=%s", baseURL, c.config.Model, c.config.APIKey)
	}

	// Default Gemini API URL format
	return fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, c.config.Model, c.config.APIKey)
}

// executeRequest performs a single HTTP request to the Gemini API.
func (c *GeminiClient) executeRequest(ctx context.Context, url string, jsonBody []byte) (*domain.AnalysisResult, error) {
	// Log request details (mask API key)
	maskedURL := maskAPIKey(url)
	c.logger.Debug("sending Gemini request",
		zap.String("url", maskedURL),
		zap.Int("body_size", len(jsonBody)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, domain.WrapError("create_request", err, false)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, domain.WrapError("gemini_timeout", domain.ErrAITimeout, true)
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
		return c.handleHTTPError(resp.StatusCode, body)
	}

	// Log raw response for debugging
	c.logger.Debug("raw Gemini response",
		zap.String("body", truncate(string(body), 2000)),
	)

	// Parse the response
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		c.logger.Warn("failed to unmarshal Gemini response",
			zap.Error(err),
			zap.String("body_preview", truncate(string(body), 500)),
		)
		return nil, domain.WrapError("parse_response", err, false)
	}

	// Check for API-level errors
	if geminiResp.Error != nil {
		return nil, domain.WrapError("gemini_api_error",
			fmt.Errorf("[%d] %s: %s", geminiResp.Error.Code, geminiResp.Error.Status, geminiResp.Error.Message), false)
	}

	// Check for blocked content
	if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
		return nil, domain.WrapError("content_blocked",
			fmt.Errorf("prompt blocked: %s", geminiResp.PromptFeedback.BlockReason), false)
	}

	// Extract the response content
	if len(geminiResp.Candidates) == 0 {
		c.logger.Warn("no candidates in response",
			zap.String("body", truncate(string(body), 1000)),
		)
		return nil, domain.WrapError("empty_response", domain.ErrInvalidAIResponse, false)
	}

	candidate := geminiResp.Candidates[0]

	c.logger.Debug("gemini candidate",
		zap.String("finish_reason", candidate.FinishReason),
		zap.Int("parts_count", len(candidate.Content.Parts)),
		zap.String("role", candidate.Content.Role),
	)

	// Check finish reason
	if candidate.FinishReason == "SAFETY" {
		return nil, domain.WrapError("safety_filter",
			fmt.Errorf("response blocked by safety filter"), false)
	}

	if len(candidate.Content.Parts) == 0 {
		c.logger.Error("empty parts in candidate - printing full response for debugging",
			zap.String("full_body", string(body)),
			zap.String("finish_reason", candidate.FinishReason),
			zap.Any("candidate", candidate),
		)
		return nil, domain.WrapError("empty_content", domain.ErrInvalidAIResponse, false)
	}

	// Extract text from parts
	var textContent strings.Builder
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textContent.WriteString(part.Text)
		}
	}

	content := textContent.String()
	if content == "" {
		return nil, domain.WrapError("empty_text", domain.ErrInvalidAIResponse, false)
	}

	// Extract and parse the JSON content from the response
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

// handleHTTPError processes HTTP error responses.
func (c *GeminiClient) handleHTTPError(statusCode int, body []byte) (*domain.AnalysisResult, error) {
	// Try to parse error response
	var errResp geminiResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		c.logger.Warn("Gemini API error",
			zap.Int("status", statusCode),
			zap.String("error_status", errResp.Error.Status),
			zap.String("error_message", errResp.Error.Message),
		)
	}

	switch statusCode {
	case http.StatusTooManyRequests:
		return nil, domain.WrapError("rate_limit", domain.ErrRateLimited, true)
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, domain.WrapError("auth_error",
			fmt.Errorf("authentication failed (status %d): check your API key", statusCode), false)
	case http.StatusBadRequest:
		return nil, domain.WrapError("bad_request",
			fmt.Errorf("bad request: %s", truncate(string(body), 200)), false)
	case http.StatusNotFound:
		return nil, domain.WrapError("model_not_found",
			fmt.Errorf("model not found: check model name in configuration"), false)
	default:
		if statusCode >= 500 {
			return nil, domain.WrapError("gemini_unavailable", domain.ErrAIUnavailable, true)
		}
		return nil, domain.WrapError("gemini_error",
			fmt.Errorf("Gemini API returned status %d: %s", statusCode, truncate(string(body), 200)), false)
	}
}

// parseAnalysisResult extracts the AnalysisResult from the Gemini response content.
func (c *GeminiClient) parseAnalysisResult(content string) (*domain.AnalysisResult, error) {
	var result domain.AnalysisResult

	// Try to find JSON in the content (Gemini might include markdown code blocks)
	jsonContent := extractJSON(content)
	if jsonContent == "" {
		c.logger.Warn("could not extract JSON from Gemini response",
			zap.String("content_preview", truncate(content, 200)),
		)
		return nil, domain.WrapError("extract_json", domain.ErrInvalidAIResponse, false)
	}

	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		c.logger.Warn("failed to unmarshal Gemini response",
			zap.Error(err),
			zap.String("json_content", truncate(jsonContent, 200)),
		)
		return nil, domain.WrapError("unmarshal_result", domain.ErrInvalidAIResponse, false)
	}

	return &result, nil
}

// HealthCheck verifies the Gemini API is reachable.
func (c *GeminiClient) HealthCheck(ctx context.Context) error {
	// Use the models.list endpoint to check connectivity
	url := fmt.Sprintf("%s/v1beta/models?key=%s", strings.TrimSuffix(c.config.BaseURL, "/"), c.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

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

// maskAPIKey masks the API key in a URL for safe logging.
func maskAPIKey(url string) string {
	if idx := strings.Index(url, "key="); idx != -1 {
		endIdx := strings.Index(url[idx:], "&")
		if endIdx == -1 {
			return url[:idx] + "key=***"
		}
		return url[:idx] + "key=***" + url[idx+endIdx:]
	}
	return url
}

// isThinkingModel returns true if the model is a thinking/reasoning model
// that uses tokens for internal reasoning (e.g., gemini-2.5-pro).
func isThinkingModel(model string) bool {
	// Gemini 2.5+ models are thinking models
	return strings.Contains(model, "2.5") ||
		strings.Contains(model, "thinking") ||
		strings.Contains(model, "reasoning")
}
