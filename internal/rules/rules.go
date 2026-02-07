// Package rules provides rule-based log pre-classification.
// Rules are applied before AI analysis to handle common, well-known patterns
// with high confidence and low latency.
package rules

import (
	"regexp"
	"strings"

	"github.com/ai-devops/internal/domain"
)

// Rule represents a single pre-classification rule.
type Rule struct {
	// ID is the unique identifier for this rule.
	ID string

	// Name is a human-readable name for the rule.
	Name string

	// Description explains what this rule detects.
	Description string

	// Patterns are regex patterns to match against log content.
	Patterns []*regexp.Regexp

	// Keywords are simple string matches (case-insensitive).
	Keywords []string

	// Confidence is the confidence level when this rule matches (0.0-1.0).
	Confidence float64

	// Result is the pre-computed analysis result.
	Result *domain.AnalysisResult
}

// Match checks if the log content matches this rule.
func (r *Rule) Match(log string) bool {
	logLower := strings.ToLower(log)

	// Check keywords first (faster)
	for _, kw := range r.Keywords {
		if strings.Contains(logLower, strings.ToLower(kw)) {
			return true
		}
	}

	// Check regex patterns
	for _, pattern := range r.Patterns {
		if pattern.MatchString(log) {
			return true
		}
	}

	return false
}

// DefaultRules returns the built-in set of rules for common log patterns.
func DefaultRules() []*Rule {
	return []*Rule{
		dockerBuildPermissionDenied(),
		dockerDaemonNotRunning(),
		npmInstallFailure(),
		outOfMemory(),
		connectionTimeout(),
		sslCertificateError(),
		diskSpaceFull(),
		portAlreadyInUse(),
		authenticationFailure(),
		kubernetesImagePullBackoff(),
	}
}

func dockerBuildPermissionDenied() *Rule {
	return &Rule{
		ID:          "docker_build_permission",
		Name:        "Docker Build Permission Denied",
		Description: "Detects Docker build failures due to permission issues",
		Keywords:    []string{"docker build", "permission denied"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)docker.*build.*permission\s+denied`),
			regexp.MustCompile(`(?i)error.*docker.*EACCES`),
		},
		Confidence: 0.9,
		Result: &domain.AnalysisResult{
			ErrorType: "docker_permission_denied",
			Severity:  domain.SeverityHigh,
			RootCause: "Docker build failed due to insufficient permissions. This typically occurs when the user running Docker doesn't have access to required files or the Docker socket.",
			SuggestedActions: []string{
				"Ensure the user is in the 'docker' group: sudo usermod -aG docker $USER",
				"Check file permissions in the build context",
				"If using CI/CD, ensure the runner has Docker socket access",
				"Verify Dockerfile COPY/ADD commands reference accessible files",
			},
			PreventionTips: []string{
				"Run Docker with appropriate user permissions",
				"Use multi-stage builds with proper ownership",
				"Configure CI/CD runners with Docker access",
			},
		},
	}
}

func dockerDaemonNotRunning() *Rule {
	return &Rule{
		ID:          "docker_daemon_not_running",
		Name:        "Docker Daemon Not Running",
		Description: "Detects when Docker daemon is not available",
		Keywords:    []string{"cannot connect to the docker daemon", "docker daemon is not running"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)cannot connect to the docker daemon`),
			regexp.MustCompile(`(?i)is the docker daemon running`),
			regexp.MustCompile(`(?i)docker\.sock.*no such file`),
		},
		Confidence: 0.95,
		Result: &domain.AnalysisResult{
			ErrorType: "docker_daemon_unavailable",
			Severity:  domain.SeverityHigh,
			RootCause: "The Docker daemon is not running or not accessible. Docker commands require a running daemon to execute.",
			SuggestedActions: []string{
				"Start the Docker daemon: sudo systemctl start docker",
				"Check Docker service status: sudo systemctl status docker",
				"Verify Docker installation: docker --version",
				"If using Docker Desktop, ensure the application is running",
			},
			PreventionTips: []string{
				"Enable Docker to start on boot: sudo systemctl enable docker",
				"Monitor Docker daemon health in production",
				"Use Docker healthchecks in CI/CD pipelines",
			},
		},
	}
}

func npmInstallFailure() *Rule {
	return &Rule{
		ID:          "npm_install_failure",
		Name:        "NPM Install Failure",
		Description: "Detects npm install failures",
		Keywords:    []string{"npm err!"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)npm ERR!.*code\s+E[A-Z]+`),
			regexp.MustCompile(`(?i)npm ERR!.*404.*not found`),
			regexp.MustCompile(`(?i)npm ERR!.*peer dep`),
		},
		Confidence: 0.85,
		Result: &domain.AnalysisResult{
			ErrorType: "npm_install_failure",
			Severity:  domain.SeverityMedium,
			RootCause: "NPM package installation failed. This could be due to missing packages, version conflicts, network issues, or corrupted cache.",
			SuggestedActions: []string{
				"Clear npm cache: npm cache clean --force",
				"Delete node_modules and package-lock.json, then reinstall",
				"Check if the package exists and version is correct",
				"Verify network connectivity to npm registry",
				"Check for peer dependency conflicts",
			},
			PreventionTips: []string{
				"Lock dependency versions in package-lock.json",
				"Use npm ci in CI/CD for reproducible builds",
				"Regularly update dependencies to avoid conflicts",
			},
		},
	}
}

func outOfMemory() *Rule {
	return &Rule{
		ID:          "out_of_memory",
		Name:        "Out of Memory",
		Description: "Detects out of memory errors",
		Keywords:    []string{"out of memory", "oomkilled", "memory allocation failed", "heap out of memory"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)out\s+of\s+memory`),
			regexp.MustCompile(`(?i)OOMKilled`),
			regexp.MustCompile(`(?i)Cannot allocate memory`),
			regexp.MustCompile(`(?i)JavaScript heap out of memory`),
			regexp.MustCompile(`(?i)java\.lang\.OutOfMemoryError`),
		},
		Confidence: 0.95,
		Result: &domain.AnalysisResult{
			ErrorType: "out_of_memory",
			Severity:  domain.SeverityHigh,
			RootCause: "The process exhausted available memory and was terminated. This can be caused by memory leaks, insufficient resource limits, or processing large datasets.",
			SuggestedActions: []string{
				"Increase memory limits for the container/process",
				"Profile the application for memory leaks",
				"Implement pagination for large data processing",
				"Check for unbounded caches or collections",
				"Review Kubernetes resource limits",
			},
			PreventionTips: []string{
				"Set appropriate memory limits based on profiling",
				"Implement memory monitoring and alerting",
				"Use streaming for large file processing",
				"Regular load testing with realistic data volumes",
			},
		},
	}
}

func connectionTimeout() *Rule {
	return &Rule{
		ID:          "connection_timeout",
		Name:        "Connection Timeout",
		Description: "Detects network-level connection timeout errors",
		Keywords:    []string{"connection timed out", "etimedout", "connection refused", "dial tcp"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)connection\s+timed?\s*out`),
			regexp.MustCompile(`(?i)ETIMEDOUT`),
			regexp.MustCompile(`(?i)ECONNREFUSED`),
			regexp.MustCompile(`(?i)dial tcp.*timeout`),
			regexp.MustCompile(`(?i)i/o timeout`),
			regexp.MustCompile(`(?i)connect:.*timeout`),
		},
		Confidence: 0.85,
		Result: &domain.AnalysisResult{
			ErrorType: "connection_timeout",
			Severity:  domain.SeverityMedium,
			RootCause: "A network connection attempt timed out. This could indicate the target service is down, network issues, firewall blocking, or incorrect host/port configuration.",
			SuggestedActions: []string{
				"Verify the target service is running and healthy",
				"Check network connectivity: ping, telnet, curl",
				"Review firewall rules and security groups",
				"Verify the host and port configuration",
				"Check DNS resolution",
			},
			PreventionTips: []string{
				"Implement health checks for dependencies",
				"Use circuit breakers for external services",
				"Configure appropriate timeout values",
				"Add retry logic with exponential backoff",
			},
		},
	}
}

func sslCertificateError() *Rule {
	return &Rule{
		ID:          "ssl_certificate_error",
		Name:        "SSL Certificate Error",
		Description: "Detects SSL/TLS certificate issues",
		Keywords:    []string{"certificate verify failed", "certificate expired"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)certificate\s+verify\s+failed`),
			regexp.MustCompile(`(?i)SSL.*certificate.*expired`),
			regexp.MustCompile(`(?i)unable to verify the first certificate`),
			regexp.MustCompile(`(?i)self.signed certificate`),
			regexp.MustCompile(`(?i)x509.*certificate`),
		},
		Confidence: 0.9,
		Result: &domain.AnalysisResult{
			ErrorType: "ssl_certificate_error",
			Severity:  domain.SeverityHigh,
			RootCause: "SSL/TLS certificate validation failed. The certificate may be expired, self-signed, issued by an untrusted CA, or the hostname doesn't match.",
			SuggestedActions: []string{
				"Check certificate expiration date",
				"Verify the certificate chain is complete",
				"Ensure the CA is trusted in the system's trust store",
				"Verify the hostname matches the certificate CN/SAN",
				"For internal services, add the CA to trusted certificates",
			},
			PreventionTips: []string{
				"Set up certificate expiration monitoring",
				"Use automated certificate renewal (Let's Encrypt)",
				"Implement certificate rotation procedures",
				"Document internal CA trust requirements",
			},
		},
	}
}

func diskSpaceFull() *Rule {
	return &Rule{
		ID:          "disk_space_full",
		Name:        "Disk Space Full",
		Description: "Detects disk space exhaustion",
		Keywords:    []string{"no space left on device", "disk full", "enospc"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)no space left on device`),
			regexp.MustCompile(`(?i)ENOSPC`),
			regexp.MustCompile(`(?i)disk\s+quota\s+exceeded`),
		},
		Confidence: 0.95,
		Result: &domain.AnalysisResult{
			ErrorType: "disk_space_full",
			Severity:  domain.SeverityHigh,
			RootCause: "The disk has run out of available space. This prevents writing new data and can cause application crashes or data corruption.",
			SuggestedActions: []string{
				"Identify large files: du -sh /* | sort -h",
				"Clean up Docker resources: docker system prune -a",
				"Remove old log files and temporary data",
				"Extend disk size if in cloud environment",
				"Check for log rotation configuration",
			},
			PreventionTips: []string{
				"Implement disk space monitoring with alerts",
				"Configure log rotation policies",
				"Set up automatic cleanup of temporary files",
				"Use separate volumes for logs and data",
			},
		},
	}
}

func portAlreadyInUse() *Rule {
	return &Rule{
		ID:          "port_in_use",
		Name:        "Port Already In Use",
		Description: "Detects port binding conflicts",
		Keywords:    []string{"address already in use", "eaddrinuse", "port is already allocated"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)address already in use`),
			regexp.MustCompile(`(?i)EADDRINUSE`),
			regexp.MustCompile(`(?i)bind.*port.*already`),
			regexp.MustCompile(`(?i)port\s+\d+.*is already allocated`),
		},
		Confidence: 0.95,
		Result: &domain.AnalysisResult{
			ErrorType: "port_already_in_use",
			Severity:  domain.SeverityMedium,
			RootCause: "The application cannot bind to the specified port because another process is already using it.",
			SuggestedActions: []string{
				"Find the process using the port: lsof -i :<port> or netstat -tlnp",
				"Stop the conflicting process or service",
				"Configure the application to use a different port",
				"Check for zombie processes from previous runs",
			},
			PreventionTips: []string{
				"Use unique ports for each service",
				"Implement graceful shutdown to release ports",
				"Use port 0 for dynamic port allocation in tests",
				"Document port assignments in project documentation",
			},
		},
	}
}

func authenticationFailure() *Rule {
	return &Rule{
		ID:          "authentication_failure",
		Name:        "Authentication Failure",
		Description: "Detects authentication and authorization failures",
		Keywords:    []string{"authentication failed", "unauthorized", "access denied", "invalid credentials"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)authentication\s+failed`),
			regexp.MustCompile(`(?i)401\s+unauthorized`),
			regexp.MustCompile(`(?i)403\s+forbidden`),
			regexp.MustCompile(`(?i)invalid\s+(credentials|token|api.?key)`),
			regexp.MustCompile(`(?i)access\s+denied`),
		},
		Confidence: 0.85,
		Result: &domain.AnalysisResult{
			ErrorType: "authentication_failure",
			Severity:  domain.SeverityHigh,
			RootCause: "Authentication or authorization failed. Credentials may be invalid, expired, or missing. The user/service may also lack required permissions.",
			SuggestedActions: []string{
				"Verify credentials are correct and not expired",
				"Check if API keys or tokens need renewal",
				"Verify the service account has required permissions",
				"Check for environment variable configuration issues",
				"Review IAM policies and role assignments",
			},
			PreventionTips: []string{
				"Use secret management systems (Vault, AWS Secrets Manager)",
				"Implement credential rotation policies",
				"Use service accounts with minimal required permissions",
				"Monitor for authentication failures in security logs",
			},
		},
	}
}

func kubernetesImagePullBackoff() *Rule {
	return &Rule{
		ID:          "k8s_image_pull_backoff",
		Name:        "Kubernetes Image Pull BackOff",
		Description: "Detects Kubernetes image pull failures",
		Keywords:    []string{"imagepullbackoff", "errimagepull", "failed to pull image"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ImagePullBackOff`),
			regexp.MustCompile(`(?i)ErrImagePull`),
			regexp.MustCompile(`(?i)failed to pull image`),
			regexp.MustCompile(`(?i)rpc error.*pulling image`),
		},
		Confidence: 0.95,
		Result: &domain.AnalysisResult{
			ErrorType: "kubernetes_image_pull_failure",
			Severity:  domain.SeverityHigh,
			RootCause: "Kubernetes cannot pull the specified container image. This could be due to image not existing, registry authentication issues, network problems, or incorrect image name/tag.",
			SuggestedActions: []string{
				"Verify the image name and tag are correct",
				"Check if the image exists in the registry",
				"Verify imagePullSecrets are configured correctly",
				"Test registry connectivity from the cluster",
				"Check if the registry requires authentication",
			},
			PreventionTips: []string{
				"Use image digests instead of mutable tags",
				"Implement CI/CD checks for image availability",
				"Configure proper registry credentials in secrets",
				"Use a container registry with high availability",
			},
		},
	}
}
