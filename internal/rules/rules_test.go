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
		// connection_timeout true positives
		{
			name:      "connection timeout - dial tcp i/o timeout",
			log:       "dial tcp 10.0.0.1:5432: i/o timeout",
			wantMatch: true,
			wantRule:  "connection_timeout",
		},
		{
			name:      "connection timeout - ETIMEDOUT",
			log:       "ETIMEDOUT: connection timed out",
			wantMatch: true,
			wantRule:  "connection_timeout",
		},
		{
			name:      "connection timeout - connection refused",
			log:       "connect: connection refused",
			wantMatch: true,
			wantRule:  "connection_timeout",
		},
		// connection_timeout true negatives (should NOT match connection_timeout)
		{
			name:      "no match - timeout budget exceeded",
			log:       "timeout budget exceeded for downstream call",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - latency threshold exceeded",
			log:       "latency threshold exceeded: 500ms > 200ms",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - context deadline exceeded",
			log:       "context deadline exceeded",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - request timeout",
			log:       "request timeout after 30s",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match",
			log:       "INFO: Application started successfully",
			wantMatch: false,
			wantRule:  "",
		},
		// npm_install_failure false positives
		{
			name:      "no match - package.json validation error",
			log:       "[ERROR] Deployer: Validation failed. The file 'package.json' is missing the 'version' field.",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - generic enoent error",
			log:       "ENOENT: no such file or directory, open '/etc/config.yaml'",
			wantMatch: false,
			wantRule:  "",
		},
		// ssl_certificate_error false positives
		{
			name:      "no match - ssl success log",
			log:       "INFO: SSL handshake completed successfully on port 443",
			wantMatch: false,
			wantRule:  "",
		},
		// port_in_use true positives
		{
			name:      "port in use - Go listen error",
			log:       "listen tcp :8080: bind: address already in use",
			wantMatch: true,
			wantRule:  "port_in_use",
		},
		{
			name:      "port in use - Error prefix",
			log:       "Error: address already in use on port 3000",
			wantMatch: true,
			wantRule:  "port_in_use",
		},
		{
			name:      "port in use - bind failed",
			log:       "bind failed: address already in use",
			wantMatch: true,
			wantRule:  "port_in_use",
		},
		{
			name:      "port in use - EADDRINUSE at start",
			log:       "EADDRINUSE: cannot bind to port 8080",
			wantMatch: true,
			wantRule:  "port_in_use",
		},
		{
			name:      "port in use - fatal EADDRINUSE",
			log:       "fatal: EADDRINUSE when starting server",
			wantMatch: true,
			wantRule:  "port_in_use",
		},
		{
			name:      "port in use - port allocation error",
			log:       "Error: port 5432 is already allocated",
			wantMatch: true,
			wantRule:  "port_in_use",
		},
		// port_in_use false positives (should NOT match)
		{
			name:      "no match - hint about address already in use",
			log:       "[INFO] ServerBoot: Application started. (Hint: If you encounter 'Address already in use' errors, check for zombie processes).",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - documentation about address in use",
			log:       "About address already in use errors: These occur when a port is occupied.",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - prevent address in use tip",
			log:       "Tip: To prevent 'address already in use' errors, use SO_REUSEADDR.",
			wantMatch: false,
			wantRule:  "",
		},
		{
			name:      "no match - conditional sentence about error",
			log:       "If you see 'address already in use', restart the service.",
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
