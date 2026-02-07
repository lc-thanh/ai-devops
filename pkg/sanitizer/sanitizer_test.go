// Package sanitizer provides unit tests for log sanitization.
package sanitizer

import (
	"strings"
	"testing"
)

func TestSanitizer_Sanitize(t *testing.T) {
	s := New(10000)

	tests := []struct {
		name          string
		input         string
		shouldContain []string
		shouldNotContain []string
	}{
		{
			name:          "mask API key",
			input:         "Error: api_key=sk-abc123xyz789secret",
			shouldNotContain: []string{"sk-abc123xyz789secret"},
		},
		{
			name:          "mask password",
			input:         "Connection failed: password=mysecretpassword123",
			shouldNotContain: []string{"mysecretpassword123"},
		},
		{
			name:          "mask bearer token",
			input:         "Authorization header: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			shouldNotContain: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		},
		{
			name:          "mask AWS access key",
			input:         "AWS Error: AKIAIOSFODNN7EXAMPLE not authorized",
			shouldNotContain: []string{"AKIAIOSFODNN7EXAMPLE"},
		},
		{
			name:          "mask GitHub token",
			input:         "git push failed: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			shouldNotContain: []string{"ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		},
		{
			name:          "preserve normal log",
			input:         "ERROR: Connection timeout to database server",
			shouldContain: []string{"ERROR", "Connection timeout", "database"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.Sanitize(tt.input)
			if err != nil {
				t.Fatalf("Sanitize() error = %v", err)
			}

			for _, should := range tt.shouldContain {
				if !strings.Contains(result, should) {
					t.Errorf("result should contain %q, got %q", should, result)
				}
			}

			for _, shouldNot := range tt.shouldNotContain {
				if strings.Contains(result, shouldNot) {
					t.Errorf("result should NOT contain %q, got %q", shouldNot, result)
				}
			}
		})
	}
}

func TestSanitizer_IsTooLarge(t *testing.T) {
	s := New(100)

	if !s.IsTooLarge(strings.Repeat("x", 101)) {
		t.Error("expected true for log > maxSize")
	}

	if s.IsTooLarge(strings.Repeat("x", 100)) {
		t.Error("expected false for log == maxSize")
	}

	if s.IsTooLarge(strings.Repeat("x", 50)) {
		t.Error("expected false for log < maxSize")
	}
}

func TestSanitizer_IsEmpty(t *testing.T) {
	s := New(1000)

	if !s.IsEmpty("") {
		t.Error("expected true for empty string")
	}

	if !s.IsEmpty("   ") {
		t.Error("expected true for whitespace only")
	}

	if !s.IsEmpty("\n\t  ") {
		t.Error("expected true for whitespace with newlines/tabs")
	}

	if s.IsEmpty("content") {
		t.Error("expected false for non-empty string")
	}
}

func TestSanitizer_Truncation(t *testing.T) {
	s := New(50)
	longLog := strings.Repeat("a", 100)

	result, _ := s.Sanitize(longLog)
	if len(result) > 50 {
		t.Errorf("result length = %d, should be <= 50", len(result))
	}
}

func TestSanitizer_SanitizeWithStats(t *testing.T) {
	s := New(1000)
	// Use patterns that definitely match the regex patterns
	input := "Error: api_key=sk-abc123xyz789secret1234567890 password=verylongpassword123"

	_, stats := s.SanitizeWithStats(input)

	if stats.OriginalSize != len(input) {
		t.Errorf("OriginalSize = %d, want %d", stats.OriginalSize, len(input))
	}

	if stats.SecretsFound < 1 {
		t.Errorf("SecretsFound = %d, want >= 1", stats.SecretsFound)
	}
}
