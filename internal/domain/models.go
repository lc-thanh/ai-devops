// Package domain contains the core domain models and types.
// These models represent the business logic contracts and are independent
// of any infrastructure concerns.
package domain

import "time"

// Severity represents the severity level of an identified issue.
type Severity string

const (
	SeverityLow    Severity = "Low"
	SeverityMedium Severity = "Medium"
	SeverityHigh   Severity = "High"
)

// IsValid checks if the severity value is one of the allowed values.
func (s Severity) IsValid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh:
		return true
	default:
		return false
	}
}

// AnalysisRequest represents an incoming log analysis request.
type AnalysisRequest struct {
	// Log is the raw log content to be analyzed.
	Log string `json:"log" binding:"required"`
}

// AnalysisResult represents the structured output of log analysis.
// This schema is enforced for all AI responses.
type AnalysisResult struct {
	// ErrorType categorizes the type of error detected.
	ErrorType string `json:"error_type"`

	// Severity indicates the impact level (Low, Medium, High).
	Severity Severity `json:"severity"`

	// RootCause describes the underlying reason for the issue.
	RootCause string `json:"root_cause"`

	// SuggestedActions lists actionable remediation steps.
	SuggestedActions []string `json:"suggested_actions"`

	// PreventionTips lists ways to prevent this issue in the future.
	PreventionTips []string `json:"prevention_tips"`
}

// AnalysisResponse wraps the analysis result with metadata.
type AnalysisResponse struct {
	// Success indicates whether the analysis completed successfully.
	Success bool `json:"success"`

	// Result contains the analysis result if successful.
	Result *AnalysisResult `json:"result,omitempty"`

	// Error contains error details if the analysis failed.
	Error string `json:"error,omitempty"`

	// Source indicates whether the result came from rules or AI.
	Source string `json:"source,omitempty"`

	// ProcessedAt is the timestamp when the analysis was completed.
	ProcessedAt time.Time `json:"processed_at"`
}

// RuleMatch represents a match from the rule-based pre-classification.
type RuleMatch struct {
	// RuleID is the unique identifier of the matched rule.
	RuleID string

	// Confidence indicates how confident the rule match is (0.0 - 1.0).
	Confidence float64

	// Result is the pre-computed analysis result from the rule.
	Result *AnalysisResult
}

// PreprocessedLog contains the log after sanitization and pre-processing.
type PreprocessedLog struct {
	// Original is the original log content before processing.
	Original string

	// Sanitized is the log content after secret masking and cleanup.
	Sanitized string

	// RuleMatches contains any matches from rule-based analysis.
	RuleMatches []RuleMatch

	// Metadata contains extracted metadata from the log.
	Metadata map[string]string
}
