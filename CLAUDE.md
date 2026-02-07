# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Run server locally
go run ./cmd/server/main.go

# Build binary
go build -o bin/server ./cmd/server

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run single package tests
go test ./internal/rules/...
go test ./pkg/sanitizer/...
```

## Environment Setup

Copy `.env.example` to `.env` before running. Key settings:
- `AI_MOCK_MODE=true` for development without API key
- `AI_API_KEY` required for production use
- `ENABLE_RULES=true` enables rule-based pre-classification

## Architecture

This is a Go API server (Gin framework) that analyzes DevOps/backend logs using a hybrid rule-based + LLM approach.

### Request Flow

```
HTTP Request -> handler/analyze.go -> service/analyzer.go
                                           |
                     +---------------------+---------------------+
                     |                                           |
              rules/engine.go                             ai/client.go
         (fast pattern matching)                    (OpenAI-compatible API)
                     |                                           |
                     +---------------------+---------------------+
                                           |
                                   pkg/sanitizer
                            (secret masking, truncation)
```

### Key Components

- **`internal/service/analyzer.go`**: Core orchestrator. Tries rules first, falls back to AI, handles AI failures with rule-based fallback.
- **`internal/ai/client.go`**: OpenAI-compatible HTTP client with retry logic and exponential backoff.
- **`internal/ai/interfaces.go`**: Defines `Client`, `PromptBuilder`, `ResponseValidator` interfaces for testability.
- **`internal/rules/`**: Pattern-matching engine with predefined rules for common DevOps errors.
- **`pkg/sanitizer/`**: Masks secrets (passwords, tokens, keys) and truncates large logs.
- **`internal/domain/models.go`**: Core types (`AnalysisResult`, `AnalysisRequest`, `Severity`).

### AI Client Pattern

The `ai.Client` interface enables swapping implementations:
- `OpenAIClient`: Production client for OpenAI-compatible APIs
- `MockClient`: Returns simulated responses for testing (enabled via `AI_MOCK_MODE=true`)

### Response Schema

All analysis results conform to `domain.AnalysisResult`:
```go
type AnalysisResult struct {
    ErrorType        string   `json:"error_type"`
    Severity         Severity `json:"severity"`        // Low|Medium|High
    RootCause        string   `json:"root_cause"`
    SuggestedActions []string `json:"suggested_actions"`
    PreventionTips   []string `json:"prevention_tips"`
}
```

## API Endpoints

- `POST /api/v1/analyze` - Main log analysis endpoint
- `POST /api/v1/ai/analyze-log` - Alias for above
- `GET /health` - Health check
- `GET /ready` - Readiness check
