// Package rules provides unit tests for the rule engine.
package rules

import (
	"testing"

	"go.uber.org/zap"
)

func TestRule_Match(t *testing.T) {
	rules := DefaultRules()

	tests := []struct {
		name      string
		log       string
		wantMatch bool
		wantRule  string
	}{
		{
			name:      "docker permission denied",
			log:       "ERROR: docker build failed: permission denied",
			wantMatch: true,
			wantRule:  "docker_build_permission",
		},
		{
			name:      "docker daemon not running",
			log:       "Cannot connect to the Docker daemon at unix:///var/run/docker.sock",
			wantMatch: true,
			wantRule:  "docker_daemon_not_running",
		},
		{
			name:      "npm install error",
			log:       "npm ERR! code ENOENT\nnpm ERR! syscall open",
			wantMatch: true,
			wantRule:  "npm_install_failure",
		},
		{
			name:      "out of memory",
			log:       "java.lang.OutOfMemoryError: Java heap space",
			wantMatch: true,
			wantRule:  "out_of_memory",
		},
		{
			name:      "kubernetes imagepullbackoff",
			log:       "Warning  Failed  pod/myapp-abc123  Failed to pull image: ErrImagePull",
			wantMatch: true,
			wantRule:  "k8s_image_pull_backoff",
		},
		{
			name:      "no match",
			log:       "INFO: Application started successfully",
			wantMatch: false,
			wantRule:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matched bool
			var matchedRuleID string

			for _, rule := range rules {
				if rule.Match(tt.log) {
					matched = true
					matchedRuleID = rule.ID
					break
				}
			}

			if matched != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", matched, tt.wantMatch)
			}

			if tt.wantMatch && matchedRuleID != tt.wantRule {
				t.Errorf("Matched rule ID = %v, want %v", matchedRuleID, tt.wantRule)
			}
		})
	}
}

func TestEngine_GetBestMatch(t *testing.T) {
	logger := zap.NewNop()
	engine := NewEngine(DefaultRules(), 0.8, logger)

	// Test with empty matches
	if best := engine.GetBestMatch(nil); best != nil {
		t.Error("expected nil for empty matches")
	}

	// Test with actual log that matches multiple rules
	// The actual behavior is tested through integration tests
}
