// Package service contains the business logic layer.
package service

import (
	"context"
	"time"

	"github.com/ai-devops/internal/ai"
	"github.com/ai-devops/internal/domain"
	"github.com/ai-devops/internal/rules"
	"github.com/ai-devops/pkg/sanitizer"
	"go.uber.org/zap"
)

// Analyzer orchestrates the log analysis pipeline.
type Analyzer struct {
	aiClient    ai.Client
	ruleEngine  *rules.Engine
	sanitizer   *sanitizer.Sanitizer
	enableRules bool
	logger      *zap.Logger
}

// AnalyzerConfig contains configuration for the Analyzer.
type AnalyzerConfig struct {
	EnableRules bool
}

// NewAnalyzer creates a new Analyzer with all dependencies.
func NewAnalyzer(
	aiClient ai.Client,
	ruleEngine *rules.Engine,
	sanitizer *sanitizer.Sanitizer,
	config AnalyzerConfig,
	logger *zap.Logger,
) *Analyzer {
	return &Analyzer{
		aiClient:    aiClient,
		ruleEngine:  ruleEngine,
		sanitizer:   sanitizer,
		enableRules: config.EnableRules,
		logger:      logger.Named("analyzer"),
	}
}

// Analyze processes a log through the analysis pipeline:
// 1. Sanitize input
// 2. Apply rule-based analysis
// 3. If no high-confidence rule match, use AI
// 4. Validate and return result
func (a *Analyzer) Analyze(ctx context.Context, req *domain.AnalysisRequest) (*domain.AnalysisResponse, error) {
	startTime := time.Now()
	a.logger.Debug("starting analysis", zap.Int("log_length", len(req.Log)))

	// Step 1: Validate input
	if a.sanitizer.IsEmpty(req.Log) {
		return &domain.AnalysisResponse{
			Success:     false,
			Error:       domain.ErrEmptyLog.Error(),
			ProcessedAt: time.Now(),
		}, nil
	}

	if a.sanitizer.IsTooLarge(req.Log) {
		a.logger.Warn("log too large, will be truncated",
			zap.Int("original_size", len(req.Log)),
		)
	}

	// Step 2: Sanitize the log
	sanitizedLog, stats := a.sanitizer.SanitizeWithStats(req.Log)
	a.logger.Debug("log sanitized",
		zap.Int("original_size", stats.OriginalSize),
		zap.Int("sanitized_size", stats.SanitizedSize),
		zap.Int("secrets_found", stats.SecretsFound),
		zap.Bool("truncated", stats.Truncated),
	)

	// Step 3: Apply rule-based analysis
	if a.enableRules {
		matches := a.ruleEngine.Analyze(sanitizedLog)
		if a.ruleEngine.ShouldUseRuleResult(matches) {
			best := a.ruleEngine.GetBestMatch(matches)
			a.logger.Info("using rule-based result",
				zap.String("rule_id", best.RuleID),
				zap.Float64("confidence", best.Confidence),
				zap.Duration("duration", time.Since(startTime)),
			)

			return &domain.AnalysisResponse{
				Success:     true,
				Result:      best.Result,
				Source:      "rules:" + best.RuleID,
				ProcessedAt: time.Now(),
			}, nil
		}

		if len(matches) > 0 {
			a.logger.Debug("rule matches below threshold, proceeding to AI",
				zap.Int("match_count", len(matches)),
			)
		}
	}

	// Step 4: Use AI for analysis
	result, err := a.aiClient.Analyze(ctx, sanitizedLog)
	if err != nil {
		a.logger.Error("AI analysis failed",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)),
		)

		// Try to use rule-based fallback if AI fails
		if a.enableRules {
			matches := a.ruleEngine.Analyze(sanitizedLog)
			if len(matches) > 0 {
				best := a.ruleEngine.GetBestMatch(matches)
				if best != nil {
					a.logger.Info("using rule-based fallback after AI failure",
						zap.String("rule_id", best.RuleID),
					)
					return &domain.AnalysisResponse{
						Success:     true,
						Result:      best.Result,
						Source:      "rules_fallback:" + best.RuleID,
						ProcessedAt: time.Now(),
					}, nil
				}
			}
		}

		return &domain.AnalysisResponse{
			Success:     false,
			Error:       err.Error(),
			ProcessedAt: time.Now(),
		}, nil
	}

	a.logger.Info("AI analysis completed",
		zap.String("error_type", result.ErrorType),
		zap.String("severity", string(result.Severity)),
		zap.Duration("duration", time.Since(startTime)),
	)

	return &domain.AnalysisResponse{
		Success:     true,
		Result:      result,
		Source:      "ai",
		ProcessedAt: time.Now(),
	}, nil
}
