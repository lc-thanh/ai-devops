// Package ai provides the AI client interface and implementations.
package ai

import (
	"fmt"

	"github.com/ai-devops/internal/domain"
)

// DefaultValidator implements ResponseValidator with strict schema checks.
type DefaultValidator struct{}

// NewDefaultValidator creates a new response validator.
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

// Validate checks if the AI response conforms to the expected schema.
func (v *DefaultValidator) Validate(result *domain.AnalysisResult) error {
	if result == nil {
		return domain.WrapError("validate", 
			fmt.Errorf("result is nil"), false)
	}

	// Validate error_type is not empty
	if result.ErrorType == "" {
		return domain.WrapError("validate_error_type",
			fmt.Errorf("%w: error_type is required", domain.ErrInvalidAIResponse), false)
	}

	// Validate severity is one of the allowed values
	if !result.Severity.IsValid() {
		return domain.WrapError("validate_severity",
			fmt.Errorf("%w: severity must be Low, Medium, or High, got: %s", 
				domain.ErrInvalidAIResponse, result.Severity), false)
	}

	// Validate root_cause is not empty
	if result.RootCause == "" {
		return domain.WrapError("validate_root_cause",
			fmt.Errorf("%w: root_cause is required", domain.ErrInvalidAIResponse), false)
	}

	// Validate suggested_actions has at least one item
	if len(result.SuggestedActions) == 0 {
		return domain.WrapError("validate_suggested_actions",
			fmt.Errorf("%w: at least one suggested_action is required", domain.ErrInvalidAIResponse), false)
	}

	// Validate each suggested_action is not empty
	for i, action := range result.SuggestedActions {
		if action == "" {
			return domain.WrapError("validate_suggested_actions",
				fmt.Errorf("%w: suggested_action[%d] is empty", domain.ErrInvalidAIResponse, i), false)
		}
	}

	// Validate each prevention_tip is not empty (if present)
	for i, tip := range result.PreventionTips {
		if tip == "" {
			return domain.WrapError("validate_prevention_tips",
				fmt.Errorf("%w: prevention_tip[%d] is empty", domain.ErrInvalidAIResponse, i), false)
		}
	}

	return nil
}
