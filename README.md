# 🤖 AI-Powered Log Analysis Service

## Overview

This project implements an **AI-Powered Log Analysis Service for DevOps**, capable of analyzing CI/CD, Docker, and application logs to identify root causes and propose actionable remediation steps. The assistant combines **rule-based detection** with **Large Language Models (LLMs)** to deliver structured, machine-consumable insights suitable for automation, dashboards, and ChatOps.

---

## Key Features

- **Log Analysis API**: Analyze raw logs and return structured diagnostics.
- **Hybrid Intelligence**: Fast rule-based checks + LLM reasoning.
- **Strict JSON Output**: Deterministic schema for easy integration.
- **Reliability Controls**: Timeout, retry, validation, and rate limiting.
- **Security-Aware**: Log sanitization and secret masking.
- **CI-Friendly**: Mock mode for testing without external API calls.

---

## API Specification

### `POST /api/v1/ai/analyze-log`

**Request**

```json
{ "log": "raw log string" }
```

**Response**

```json
{
  "error_type": "string",
  "severity": "Low|Medium|High",
  "root_cause": "string",
  "suggested_actions": ["string"],
  "prevention_tips": ["string"]
}
```

---

## Architecture

```
Client (CLI / Web / ChatOps)
        ↓
Golang Backend (API)
        ↓
Log Preprocessor (rules + sanitize)
        ↓
AI Service (LLM client + prompts)
        ↓
JSON Validator
        ↓
Response / Optional History Store
```

---

## Quick Start

### 1. Clone repository

```bash
git clone <repo-url>
cd <repo-name>
```

### 2. Environment configuration

```bash
cp .env.example .env
```

```env
PORT=8080
AI_API_KEY=your_api_key_here
AI_MODEL=gpt-4o-mini
AI_TIMEOUT_SECONDS=8
AI_MAX_TOKENS=512
```

### 3. Run locally

```bash
go run ./cmd/server/main.go
```

### 4. Example request

```bash
curl -X POST http://localhost:8080/api/v1/ai/analyze-log   -H "Content-Type: application/json"   -d '{"log":"ERROR: docker build failed: permission denied"}'
```

---

## Prompting Strategy

**System Prompt**

```
You are a senior DevOps engineer diagnosing CI/CD, Docker,
and backend system logs. Provide concise root cause analysis
and actionable remediation steps. Return strictly valid JSON
matching the provided schema.
```

**User Prompt Template**

```
Analyze the following log and return valid JSON exactly matching
this schema:
{ "error_type": "", "severity": "Low|Medium|High",
  "root_cause": "", "suggested_actions": [],
  "prevention_tips": [] }

Log:
---
{{LOG_CONTENT}}
---
```

---

## Engineering Considerations

- **Timeout & Retry**: Single retry with exponential backoff.
- **Validation**: JSON unmarshal enforcement; fallback on failure.
- **Sanitization**: Mask secrets (passwords, tokens, keys).
- **Rate Limiting**: Prevent runaway LLM usage.
- **Observability**: Log latency and response size only.

---

## Testing

- Unit tests with mocked LLM client.
- Handler tests using `httptest`.
- CI runs without requiring `AI_API_KEY`.

---

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **AI**: AI-compatible LLM API
- **CI/CD**: GitHub Actions
- **Container**: Docker

---

## Roadmap

- Retrieval-Augmented Generation (RAG) with vector databases.
- Incident analytics dashboard.
- ChatOps integration (Slack / Teams).
- Cost monitoring and prompt evaluation.

---

Happy debugging 🚀
