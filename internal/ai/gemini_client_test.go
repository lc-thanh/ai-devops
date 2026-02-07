// Package ai provides unit tests for the Gemini client.
package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-devops/internal/config"
	"github.com/ai-devops/internal/domain"
	"go.uber.org/zap"
)

func TestGeminiClient_Analyze(t *testing.T) {
	logger := zap.NewNop()
	prompter, _ := NewDefaultPromptBuilder()
	validator := NewDefaultValidator()

	tests := []struct {
		name        string
		response    geminiResponse
		statusCode  int
		wantErr     bool
		errContains string
	}{
		{
			name: "successful response",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role: "model",
							Parts: []geminiPart{
								{Text: `{"error_type":"docker_build_failure","severity":"High","root_cause":"Missing base image","suggested_actions":["Pull the image"],"prevention_tips":["Use pinned versions"]}`},
							},
						},
						FinishReason: "STOP",
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "response with markdown code block",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role: "model",
							Parts: []geminiPart{
								{Text: "```json\n{\"error_type\":\"permission_denied\",\"severity\":\"Medium\",\"root_cause\":\"No access\",\"suggested_actions\":[\"Grant access\"],\"prevention_tips\":[\"Check permissions\"]}\n```"},
							},
						},
						FinishReason: "STOP",
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "rate limited",
			response:   geminiResponse{},
			statusCode: http.StatusTooManyRequests,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			response:   geminiResponse{},
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "server error",
			response:   geminiResponse{},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name: "empty candidates",
			response: geminiResponse{
				Candidates: []geminiCandidate{},
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name: "blocked by safety filter",
			response: geminiResponse{
				Candidates: []geminiCandidate{
					{
						Content:      geminiContent{},
						FinishReason: "SAFETY",
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name: "prompt blocked",
			response: geminiResponse{
				PromptFeedback: &geminiPromptFeedback{
					BlockReason: "SAFETY",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name: "API error in response",
			response: geminiResponse{
				Error: &geminiError{
					Code:    400,
					Message: "Invalid request",
					Status:  "INVALID_ARGUMENT",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and content type
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json")
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			cfg := &config.AIConfig{
				Provider:   config.AIProviderGemini,
				APIKey:     "test-api-key",
				BaseURL:    server.URL,
				Model:      "gemini-2.0-flash",
				Timeout:    5 * time.Second,
				MaxTokens:  512,
				MaxRetries: 0,
			}

			client := NewGeminiClient(cfg, prompter, validator, logger)
			result, err := client.Analyze(context.Background(), "test log content")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("expected result, got nil")
				return
			}

			if result.ErrorType == "" {
				t.Error("error_type should not be empty")
			}
		})
	}
}

func TestGeminiClient_HealthCheck(t *testing.T) {
	logger := zap.NewNop()
	prompter, _ := NewDefaultPromptBuilder()
	validator := NewDefaultValidator()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "healthy",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unhealthy",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := &config.AIConfig{
				Provider: config.AIProviderGemini,
				APIKey:   "test-api-key",
				BaseURL:  server.URL,
				Timeout:  5 * time.Second,
			}

			client := NewGeminiClient(cfg, prompter, validator, logger)
			err := client.HealthCheck(context.Background())

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGeminiClient_BuildURL(t *testing.T) {
	logger := zap.NewNop()
	prompter, _ := NewDefaultPromptBuilder()
	validator := NewDefaultValidator()

	tests := []struct {
		name     string
		baseURL  string
		model    string
		apiKey   string
		expected string
	}{
		{
			name:     "default base URL",
			baseURL:  "https://generativelanguage.googleapis.com",
			model:    "gemini-2.0-flash",
			apiKey:   "test-key",
			expected: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=test-key",
		},
		{
			name:     "base URL with version",
			baseURL:  "https://generativelanguage.googleapis.com/v1",
			model:    "gemini-1.5-pro",
			apiKey:   "test-key",
			expected: "https://generativelanguage.googleapis.com/v1/models/gemini-1.5-pro:generateContent?key=test-key",
		},
		{
			name:     "trailing slash removed",
			baseURL:  "https://generativelanguage.googleapis.com/",
			model:    "gemini-2.0-flash",
			apiKey:   "my-key",
			expected: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=my-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AIConfig{
				Provider: config.AIProviderGemini,
				APIKey:   tt.apiKey,
				BaseURL:  tt.baseURL,
				Model:    tt.model,
				Timeout:  5 * time.Second,
			}

			client := NewGeminiClient(cfg, prompter, validator, logger)
			url := client.buildURL()

			if url != tt.expected {
				t.Errorf("buildURL() = %s, want %s", url, tt.expected)
			}
		})
	}
}

func TestGeminiClient_ParseAnalysisResult(t *testing.T) {
	logger := zap.NewNop()
	prompter, _ := NewDefaultPromptBuilder()
	validator := NewDefaultValidator()

	cfg := &config.AIConfig{
		Provider: config.AIProviderGemini,
		APIKey:   "test-key",
		BaseURL:  "https://test.com",
		Model:    "gemini-2.0-flash",
		Timeout:  5 * time.Second,
	}

	client := NewGeminiClient(cfg, prompter, validator, logger)

	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantType  string
	}{
		{
			name:     "pure JSON",
			content:  `{"error_type":"test","severity":"High","root_cause":"cause","suggested_actions":["fix"],"prevention_tips":["prevent"]}`,
			wantErr:  false,
			wantType: "test",
		},
		{
			name:     "JSON in markdown",
			content:  "```json\n{\"error_type\":\"test2\",\"severity\":\"Low\",\"root_cause\":\"cause\",\"suggested_actions\":[\"fix\"],\"prevention_tips\":[]}\n```",
			wantErr:  false,
			wantType: "test2",
		},
		{
			name:    "no JSON",
			content: "This is plain text without any JSON",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			content: "{error_type: test}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseAnalysisResult(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.ErrorType != tt.wantType {
				t.Errorf("error_type = %s, want %s", result.ErrorType, tt.wantType)
			}
		})
	}
}

func TestGeminiClient_HandleHTTPError(t *testing.T) {
	logger := zap.NewNop()
	prompter, _ := NewDefaultPromptBuilder()
	validator := NewDefaultValidator()

	cfg := &config.AIConfig{
		Provider: config.AIProviderGemini,
		APIKey:   "test-key",
		BaseURL:  "https://test.com",
		Model:    "gemini-2.0-flash",
		Timeout:  5 * time.Second,
	}

	client := NewGeminiClient(cfg, prompter, validator, logger)

	tests := []struct {
		name       string
		statusCode int
		body       []byte
		retryable  bool
	}{
		{
			name:       "rate limit is retryable",
			statusCode: http.StatusTooManyRequests,
			body:       []byte("rate limited"),
			retryable:  true,
		},
		{
			name:       "auth error is not retryable",
			statusCode: http.StatusUnauthorized,
			body:       []byte("unauthorized"),
			retryable:  false,
		},
		{
			name:       "server error is retryable",
			statusCode: http.StatusInternalServerError,
			body:       []byte("server error"),
			retryable:  true,
		},
		{
			name:       "bad request is not retryable",
			statusCode: http.StatusBadRequest,
			body:       []byte("bad request"),
			retryable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.handleHTTPError(tt.statusCode, tt.body)

			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			if domain.IsRetryable(err) != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", domain.IsRetryable(err), tt.retryable)
			}
		})
	}
}
