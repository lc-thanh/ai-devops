// Package handler contains HTTP handlers for the API.
package handler

import (
	"net/http"
	"time"

	"github.com/ai-devops/internal/domain"
	"github.com/ai-devops/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AnalyzeHandler handles log analysis requests.
type AnalyzeHandler struct {
	analyzer *service.Analyzer
	logger   *zap.Logger
}

// NewAnalyzeHandler creates a new AnalyzeHandler.
func NewAnalyzeHandler(analyzer *service.Analyzer, logger *zap.Logger) *AnalyzeHandler {
	return &AnalyzeHandler{
		analyzer: analyzer,
		logger:   logger.Named("analyze_handler"),
	}
}

// Handle processes POST /analyze requests.
func (h *AnalyzeHandler) Handle(c *gin.Context) {
	startTime := time.Now()
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	logger := h.logger.With(zap.String("request_id", requestID))
	logger.Debug("received analysis request")

	// Parse request body
	var req domain.AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, domain.AnalysisResponse{
			Success:     false,
			Error:       "Invalid request body: " + err.Error(),
			ProcessedAt: time.Now(),
		})
		return
	}

	// Perform analysis
	ctx := c.Request.Context()
	response, err := h.analyzer.Analyze(ctx, &req)
	if err != nil {
		logger.Error("analysis failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, domain.AnalysisResponse{
			Success:     false,
			Error:       "Internal error during analysis",
			ProcessedAt: time.Now(),
		})
		return
	}

	// Log completion
	logger.Info("analysis completed",
		zap.Bool("success", response.Success),
		zap.String("source", response.Source),
		zap.Duration("duration", time.Since(startTime)),
	)

	// Return appropriate status code
	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusUnprocessableEntity, response)
	}
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	logger *zap.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger.Named("health_handler"),
	}
}

// Handle processes GET /health requests.
func (h *HealthHandler) Handle(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadyHandler handles readiness check requests.
type ReadyHandler struct {
	logger *zap.Logger
}

// NewReadyHandler creates a new ReadyHandler.
func NewReadyHandler(logger *zap.Logger) *ReadyHandler {
	return &ReadyHandler{
		logger: logger.Named("ready_handler"),
	}
}

// Handle processes GET /ready requests.
func (h *ReadyHandler) Handle(c *gin.Context) {
	// TODO: Add actual readiness checks (e.g., AI service health)
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// generateRequestID creates a simple unique request ID.
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}
