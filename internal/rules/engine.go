// Package rules provides rule-based log pre-classification.
package rules

import (
	"github.com/ai-devops/internal/domain"
	"go.uber.org/zap"
)

// Engine applies rules to logs before AI analysis.
type Engine struct {
	rules               []*Rule
	confidenceThreshold float64
	logger              *zap.Logger
}

// NewEngine creates a new rule engine with the provided configuration.
func NewEngine(rules []*Rule, confidenceThreshold float64, logger *zap.Logger) *Engine {
	return &Engine{
		rules:               rules,
		confidenceThreshold: confidenceThreshold,
		logger:              logger.Named("rule_engine"),
	}
}

// Analyze applies all rules to the log and returns matches.
func (e *Engine) Analyze(log string) []domain.RuleMatch {
	var matches []domain.RuleMatch

	for _, rule := range e.rules {
		if rule.Match(log) {
			e.logger.Debug("rule matched",
				zap.String("rule_id", rule.ID),
				zap.Float64("confidence", rule.Confidence),
			)

			matches = append(matches, domain.RuleMatch{
				RuleID:     rule.ID,
				Confidence: rule.Confidence,
				Result:     rule.Result,
			})
		}
	}

	return matches
}

// GetBestMatch returns the highest confidence match that exceeds the threshold.
// Returns nil if no match exceeds the threshold.
func (e *Engine) GetBestMatch(matches []domain.RuleMatch) *domain.RuleMatch {
	if len(matches) == 0 {
		return nil
	}

	var best *domain.RuleMatch
	for i := range matches {
		match := &matches[i]
		if match.Confidence >= e.confidenceThreshold {
			if best == nil || match.Confidence > best.Confidence {
				best = match
			}
		}
	}

	return best
}

// ShouldUseRuleResult determines if a rule result should be used instead of AI.
func (e *Engine) ShouldUseRuleResult(matches []domain.RuleMatch) bool {
	best := e.GetBestMatch(matches)
	return best != nil && best.Confidence >= e.confidenceThreshold
}
