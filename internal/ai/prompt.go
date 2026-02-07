// Package ai provides the AI client interface and implementations.
package ai

import (
	"bytes"
	"text/template"
)

// DefaultPromptBuilder implements PromptBuilder with templated prompts.
type DefaultPromptBuilder struct {
	systemPrompt   string
	userTemplate   *template.Template
}

// systemPromptText defines the AI's role and behavior.
// This prompt is versioned as code and can be reviewed/tested.
const systemPromptText = `You are a senior DevOps engineer diagnosing CI/CD, Docker, Kubernetes, and backend system logs.

Your responsibilities:
1. Identify the type of error and categorize it appropriately
2. Determine the severity (Low, Medium, High) based on impact
3. Provide a clear, concise root cause analysis
4. Suggest specific, actionable remediation steps
5. Recommend prevention strategies for the future

Guidelines:
- Be specific and technical in your analysis
- Focus on actionable insights, not general advice
- Consider common DevOps patterns and anti-patterns
- Reference specific technologies when applicable
- Severity levels:
  - High: Production outages, security vulnerabilities, data loss risks
  - Medium: Performance degradation, partial failures, deprecated usage
  - Low: Warnings, style issues, minor configuration problems

CRITICAL: You MUST respond with ONLY valid JSON matching the exact schema provided. No markdown, no explanations, just the JSON object.`

// userPromptTemplate defines how log content is presented to the AI.
const userPromptTemplate = `Analyze the following log and return valid JSON exactly matching this schema:

{
  "error_type": "string - category of the error (e.g., 'docker_build_failure', 'permission_denied', 'connection_timeout')",
  "severity": "Low|Medium|High",
  "root_cause": "string - concise explanation of why this error occurred",
  "suggested_actions": ["string array - specific steps to fix the issue"],
  "prevention_tips": ["string array - how to prevent this in the future"]
}

Log content:
---
{{.Log}}
---

Respond with ONLY the JSON object, no additional text.`

// NewDefaultPromptBuilder creates a new prompt builder with default templates.
func NewDefaultPromptBuilder() (*DefaultPromptBuilder, error) {
	tmpl, err := template.New("user_prompt").Parse(userPromptTemplate)
	if err != nil {
		return nil, err
	}

	return &DefaultPromptBuilder{
		systemPrompt: systemPromptText,
		userTemplate: tmpl,
	}, nil
}

// BuildSystemPrompt returns the system prompt.
func (p *DefaultPromptBuilder) BuildSystemPrompt() string {
	return p.systemPrompt
}

// BuildUserPrompt constructs the user prompt with the log content.
func (p *DefaultPromptBuilder) BuildUserPrompt(log string) string {
	var buf bytes.Buffer
	data := struct {
		Log string
	}{
		Log: log,
	}

	if err := p.userTemplate.Execute(&buf, data); err != nil {
		// Fallback to simple format if template fails
		return "Analyze this log and return JSON:\n\n" + log
	}

	return buf.String()
}

// CustomPromptBuilder allows for custom prompt configurations.
type CustomPromptBuilder struct {
	systemPrompt string
	userTemplate *template.Template
}

// NewCustomPromptBuilder creates a prompt builder with custom templates.
func NewCustomPromptBuilder(systemPrompt, userPromptTmpl string) (*CustomPromptBuilder, error) {
	tmpl, err := template.New("custom_user_prompt").Parse(userPromptTmpl)
	if err != nil {
		return nil, err
	}

	return &CustomPromptBuilder{
		systemPrompt: systemPrompt,
		userTemplate: tmpl,
	}, nil
}

// BuildSystemPrompt returns the custom system prompt.
func (p *CustomPromptBuilder) BuildSystemPrompt() string {
	return p.systemPrompt
}

// BuildUserPrompt constructs the user prompt with the log content.
func (p *CustomPromptBuilder) BuildUserPrompt(log string) string {
	var buf bytes.Buffer
	data := struct {
		Log string
	}{
		Log: log,
	}

	if err := p.userTemplate.Execute(&buf, data); err != nil {
		return "Analyze this log:\n\n" + log
	}

	return buf.String()
}
