// Package ai provides unit tests for the AI client.
package ai

import (
	"testing"

	"github.com/ai-devops/internal/domain"
)

func TestDefaultValidator_Validate(t *testing.T) {
	v := NewDefaultValidator()

	tests := []struct {
		name    string
		result  *domain.AnalysisResult
		wantErr bool
	}{
		{
			name: "valid result",
			result: &domain.AnalysisResult{
				ErrorType:        "test_error",
				Severity:         domain.SeverityHigh,
				RootCause:        "Test root cause",
				SuggestedActions: []string{"Fix it"},
				PreventionTips:   []string{"Don't break it"},
			},
			wantErr: false,
		},
		{
			name:    "nil result",
			result:  nil,
			wantErr: true,
		},
		{
			name: "empty error type",
			result: &domain.AnalysisResult{
				ErrorType:        "",
				Severity:         domain.SeverityHigh,
				RootCause:        "Test root cause",
				SuggestedActions: []string{"Fix it"},
			},
			wantErr: true,
		},
		{
			name: "invalid severity",
			result: &domain.AnalysisResult{
				ErrorType:        "test_error",
				Severity:         "Invalid",
				RootCause:        "Test root cause",
				SuggestedActions: []string{"Fix it"},
			},
			wantErr: true,
		},
		{
			name: "empty root cause",
			result: &domain.AnalysisResult{
				ErrorType:        "test_error",
				Severity:         domain.SeverityHigh,
				RootCause:        "",
				SuggestedActions: []string{"Fix it"},
			},
			wantErr: true,
		},
		{
			name: "empty suggested actions",
			result: &domain.AnalysisResult{
				ErrorType:        "test_error",
				Severity:         domain.SeverityHigh,
				RootCause:        "Test root cause",
				SuggestedActions: []string{},
			},
			wantErr: true,
		},
		{
			name: "empty action in list",
			result: &domain.AnalysisResult{
				ErrorType:        "test_error",
				Severity:         domain.SeverityHigh,
				RootCause:        "Test root cause",
				SuggestedActions: []string{"Valid", ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantJSON bool
	}{
		{
			name:     "pure JSON",
			content:  `{"error_type": "test"}`,
			wantJSON: true,
		},
		{
			name:     "JSON in markdown",
			content:  "```json\n{\"error_type\": \"test\"}\n```",
			wantJSON: true,
		},
		{
			name:     "JSON with prefix text",
			content:  "Here is the analysis:\n{\"error_type\": \"test\"}",
			wantJSON: true,
		},
		{
			name:     "no JSON",
			content:  "This is just plain text",
			wantJSON: false,
		},
		{
			name:     "invalid JSON",
			content:  "{error_type: test}",
			wantJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.content)
			gotJSON := result != ""
			if gotJSON != tt.wantJSON {
				t.Errorf("extractJSON() got JSON = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

func TestDefaultPromptBuilder(t *testing.T) {
	builder, err := NewDefaultPromptBuilder()
	if err != nil {
		t.Fatalf("failed to create prompt builder: %v", err)
	}

	// Test system prompt
	sysPrompt := builder.BuildSystemPrompt()
	if sysPrompt == "" {
		t.Error("system prompt should not be empty")
	}

	// Test user prompt
	testLog := "ERROR: something went wrong"
	userPrompt := builder.BuildUserPrompt(testLog)
	if userPrompt == "" {
		t.Error("user prompt should not be empty")
	}
	if !contains(userPrompt, testLog) {
		t.Error("user prompt should contain the log")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
