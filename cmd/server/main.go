// AI DevOps Assistant - Server Entry Point
//
// This is the main entry point for the AI-powered DevOps assistant.
// It initializes all dependencies and starts the HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-devops/internal/ai"
	"github.com/ai-devops/internal/config"
	"github.com/ai-devops/internal/handler"
	"github.com/ai-devops/internal/logger"
	"github.com/ai-devops/internal/rules"
	"github.com/ai-devops/internal/service"
	"github.com/ai-devops/pkg/sanitizer"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load .env file if it exists (development)
	_ = godotenv.Load()

	// Determine if we're in development mode
	isDev := os.Getenv("GIN_MODE") != "release"

	// Initialize logger
	zapLogger, err := logger.New(isDev)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("starting AI DevOps Assistant",
		zap.Bool("development", isDev),
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		zapLogger.Fatal("failed to load configuration", zap.Error(err))
	}

	zapLogger.Info("configuration loaded",
		zap.String("port", cfg.Server.Port),
		zap.String("ai_model", cfg.AI.Model),
		zap.Bool("mock_mode", cfg.AI.MockMode),
		zap.Bool("rules_enabled", cfg.Processing.EnableRules),
	)

	// Initialize dependencies
	var aiClient ai.Client
	if cfg.AI.MockMode {
		zapLogger.Warn("running in mock mode - AI responses are simulated")
		aiClient = ai.NewMockClient(zapLogger)
	} else {
		// Create prompt builder
		promptBuilder, err := ai.NewDefaultPromptBuilder()
		if err != nil {
			zapLogger.Fatal("failed to create prompt builder", zap.Error(err))
		}

		// Create validator
		validator := ai.NewDefaultValidator()

		// Create OpenAI client
		aiClient = ai.NewOpenAIClient(&cfg.AI, promptBuilder, validator, zapLogger)
	}

	// Initialize rule engine
	ruleEngine := rules.NewEngine(
		rules.DefaultRules(),
		cfg.Processing.RuleConfidenceThreshold,
		zapLogger,
	)

	// Initialize sanitizer
	logSanitizer := sanitizer.New(cfg.Processing.MaxLogSize)

	// Initialize analyzer service
	analyzerSvc := service.NewAnalyzer(
		aiClient,
		ruleEngine,
		logSanitizer,
		service.AnalyzerConfig{
			EnableRules: cfg.Processing.EnableRules,
		},
		zapLogger,
	)

	// Initialize handlers
	analyzeHandler := handler.NewAnalyzeHandler(analyzerSvc, zapLogger)
	healthHandler := handler.NewHealthHandler(zapLogger)
	readyHandler := handler.NewReadyHandler(zapLogger)

	// Setup Gin router
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply middleware
	router.Use(handler.RecoveryMiddleware(zapLogger))
	router.Use(handler.RequestIDMiddleware())
	router.Use(handler.LoggingMiddleware(zapLogger))
	router.Use(handler.CORSMiddleware())

	// Register routes
	router.GET("/health", healthHandler.Handle)
	router.GET("/ready", readyHandler.Handle)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/analyze", analyzeHandler.Handle)
		// Alias for the README spec
		v1.POST("/ai/analyze-log", analyzeHandler.Handle)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		zapLogger.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down server...")

	// Give the server 10 seconds to finish processing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("server stopped")
}
