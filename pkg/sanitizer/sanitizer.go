// Package sanitizer provides log sanitization and secret masking.
package sanitizer

import (
	"regexp"
	"strings"
)

// Sanitizer handles log preprocessing and secret masking.
type Sanitizer struct {
	patterns []*regexp.Regexp
	maxSize  int
}

// Pattern definitions for common secrets and sensitive data.
var defaultPatterns = []*regexp.Regexp{
	// API Keys (generic patterns)
	regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?([a-zA-Z0-9_\-]{20,})['"]?`),
	regexp.MustCompile(`(?i)(secret[_-]?key|secretkey)\s*[:=]\s*['"]?([a-zA-Z0-9_\-]{20,})['"]?`),
	regexp.MustCompile(`(?i)(access[_-]?key|accesskey)\s*[:=]\s*['"]?([a-zA-Z0-9_\-]{16,})['"]?`),

	// Authentication tokens
	regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9_\-\.]+`),
	regexp.MustCompile(`(?i)(authorization:\s*)[a-zA-Z0-9_\-\.\s]+`),
	regexp.MustCompile(`(?i)(token|auth[_-]?token)\s*[:=]\s*['"]?([a-zA-Z0-9_\-\.]{20,})['"]?`),

	// Passwords
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*['"]?([^\s'"]{4,})['"]?`),

	// AWS credentials
	regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`(?i)(aws[_-]?secret[_-]?access[_-]?key)\s*[:=]\s*['"]?([a-zA-Z0-9/+=]{40})['"]?`),

	// Private keys
	regexp.MustCompile(`-----BEGIN\s+(RSA|DSA|EC|OPENSSH)?\s*PRIVATE KEY-----`),
	regexp.MustCompile(`-----BEGIN\s+PGP\s+PRIVATE\s+KEY\s+BLOCK-----`),

	// Database connection strings
	regexp.MustCompile(`(?i)(mongodb|mysql|postgres|postgresql|redis):\/\/[^@]+@[^\s]+`),
	regexp.MustCompile(`(?i)(connection[_-]?string)\s*[:=]\s*['"]?([^\s'"]+)['"]?`),

	// GitHub tokens
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`ghu_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`ghs_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`ghr_[a-zA-Z0-9]{36}`),

	// JWT tokens
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),

	// Slack tokens
	regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z-]+`),

	// Generic high-entropy strings that look like secrets
	regexp.MustCompile(`(?i)(secret|private|credential)\s*[:=]\s*['"]?([a-zA-Z0-9_\-]{16,})['"]?`),

	// IP addresses with ports (might be internal infrastructure)
	regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}:\d{4,5}\b`),

	// Email addresses (PII)
	regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
}

// New creates a new Sanitizer with default patterns.
func New(maxSize int) *Sanitizer {
	return &Sanitizer{
		patterns: defaultPatterns,
		maxSize:  maxSize,
	}
}

// NewWithPatterns creates a Sanitizer with custom patterns.
func NewWithPatterns(maxSize int, patterns []*regexp.Regexp) *Sanitizer {
	return &Sanitizer{
		patterns: patterns,
		maxSize:  maxSize,
	}
}

// Sanitize processes the log, masking secrets and enforcing size limits.
func (s *Sanitizer) Sanitize(log string) (string, error) {
	// Trim whitespace
	log = strings.TrimSpace(log)

	// Enforce size limit
	if len(log) > s.maxSize {
		log = log[:s.maxSize]
	}

	// Mask secrets
	sanitized := s.maskSecrets(log)

	return sanitized, nil
}

// maskSecrets replaces sensitive patterns with masked versions.
func (s *Sanitizer) maskSecrets(log string) string {
	result := log

	for _, pattern := range s.patterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return maskValue(match)
		})
	}

	return result
}

// maskValue creates a masked version of a matched secret.
func maskValue(match string) string {
	// Preserve some context while masking the actual value
	if len(match) <= 8 {
		return "[REDACTED]"
	}

	// For longer matches, show first and last few characters
	// This helps with debugging while protecting the actual secret
	if idx := strings.IndexAny(match, ":="); idx != -1 {
		prefix := match[:idx+1]
		return prefix + "[REDACTED]"
	}

	// For tokens and keys, show format but redact content
	if len(match) > 10 {
		return match[:4] + "****" + match[len(match)-4:]
	}

	return "[REDACTED]"
}

// IsEmpty checks if the log is empty or whitespace only.
func (s *Sanitizer) IsEmpty(log string) bool {
	return strings.TrimSpace(log) == ""
}

// IsTooLarge checks if the log exceeds the maximum size.
func (s *Sanitizer) IsTooLarge(log string) bool {
	return len(log) > s.maxSize
}

// GetStats returns statistics about the sanitization.
type SanitizationStats struct {
	OriginalSize  int
	SanitizedSize int
	Truncated     bool
	SecretsFound  int
}

// SanitizeWithStats performs sanitization and returns statistics.
func (s *Sanitizer) SanitizeWithStats(log string) (string, SanitizationStats) {
	stats := SanitizationStats{
		OriginalSize: len(log),
		Truncated:    len(log) > s.maxSize,
	}

	// Count secrets before masking
	for _, pattern := range s.patterns {
		matches := pattern.FindAllString(log, -1)
		stats.SecretsFound += len(matches)
	}

	sanitized, _ := s.Sanitize(log)
	stats.SanitizedSize = len(sanitized)

	return sanitized, stats
}
