package security

import (
	"regexp"
	"strings"
)

const RedactedPlaceholder = "***REDACTED***"

var (
	// Matches uppercase environment variable assignments, e.g. OPENAI_API_KEY=..., SECRET_TOKEN=...
	envAssignRegex = regexp.MustCompile(`\b([A-Z0-9_]*(?:KEY|TOKEN|SECRET|PASSWORD|AUTH|CREDENTIALS)[A-Z0-9_]*\s*=\s*)([^\s"';&]+)`)

	// Matches JSON, YAML, or query style key-value pairs
	jsonYamlKeyRegex = regexp.MustCompile(`(?i)(["']?(?:api[_-]?key|access[_-]?token|auth[_-]?token|secret[_-]?key|client[_-]?secret|password)["']?\s*[:=]\s*["']?)([^"'\s,]+)(["']?)`)

	// Matches CLI flags like --api-key=..., --token=..., --secret ...
	cliFlagRegex = regexp.MustCompile(`(?i)(--(?:api[_-]?key|token|auth[_-]?token|secret|password)(?:=|\s+))([^\s"']+)`)

	// Matches HTTP Authorization Bearer tokens
	bearerRegex = regexp.MustCompile(`(?i)\b(bearer\s+)([a-zA-Z0-9_\-\.]{15,})\b`)

	// Matches specific known API key prefix patterns
	anthropicRegex = regexp.MustCompile(`\bsk-ant-[a-zA-Z0-9_-]{20,}\b`)
	deepseekRegex  = regexp.MustCompile(`\bsk-[a-f0-9]{32,}\b`)
	openAIRegex    = regexp.MustCompile(`\bsk-(?:proj-|svcacct-|none-)?[a-zA-Z0-9_-]{20,}\b`)
	genericSkRegex = regexp.MustCompile(`\bsk-[a-zA-Z0-9]{24,}\b`)
	ghTokenRegex   = regexp.MustCompile(`\bgh[pousr]-[a-zA-Z0-9]{20,}\b`)
	hfTokenRegex   = regexp.MustCompile(`\bhf_[a-zA-Z0-9]{20,}\b`)
)

// SanitizeText scans string content and replaces any sensitive credentials,
// API keys, Bearer tokens, or secret environment variables with ***REDACTED***.
func SanitizeText(s string) string {
	if s == "" {
		return ""
	}

	res := envAssignRegex.ReplaceAllString(s, "${1}"+RedactedPlaceholder)
	res = jsonYamlKeyRegex.ReplaceAllString(res, "${1}"+RedactedPlaceholder+"${3}")
	res = cliFlagRegex.ReplaceAllString(res, "${1}"+RedactedPlaceholder)
	res = bearerRegex.ReplaceAllString(res, "${1}"+RedactedPlaceholder)
	res = anthropicRegex.ReplaceAllString(res, RedactedPlaceholder)
	res = deepseekRegex.ReplaceAllString(res, RedactedPlaceholder)
	res = openAIRegex.ReplaceAllString(res, RedactedPlaceholder)
	res = genericSkRegex.ReplaceAllString(res, RedactedPlaceholder)
	res = ghTokenRegex.ReplaceAllString(res, RedactedPlaceholder)
	res = hfTokenRegex.ReplaceAllString(res, RedactedPlaceholder)
	return res
}

// SanitizeBytes converts byte slice to sanitized string bytes.
func SanitizeBytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	return []byte(SanitizeText(string(b)))
}

// SanitizeArgs scrubs command line argument slices, masking any keys or flags.
func SanitizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	sanitized := make([]string, len(args))
	maskNext := false

	sensitiveFlagNames := map[string]bool{
		"--api-key":       true,
		"-k":              true,
		"--token":         true,
		"--auth-token":    true,
		"--secret":        true,
		"--password":      true,
		"--openai-key":    true,
		"--anthropic-key": true,
	}

	for i, arg := range args {
		if maskNext {
			sanitized[i] = RedactedPlaceholder
			maskNext = false
			continue
		}

		lower := strings.ToLower(arg)
		if sensitiveFlagNames[lower] {
			sanitized[i] = arg
			maskNext = true
			continue
		}

		sanitized[i] = SanitizeText(arg)
	}

	return sanitized
}

// SanitizeEnv scrubs environment variables in KEY=VALUE format.
func SanitizeEnv(env []string) []string {
	if len(env) == 0 {
		return env
	}

	sanitized := make([]string, len(env))
	sensitiveKeySuffixes := []string{"_KEY", "_TOKEN", "_SECRET", "_PASSWORD", "_AUTH", "_CREDENTIALS"}

	for i, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			sanitized[i] = SanitizeText(entry)
			continue
		}

		key := parts[0]
		upperKey := strings.ToUpper(key)
		isSensitive := false

		for _, suffix := range sensitiveKeySuffixes {
			if strings.HasSuffix(upperKey, suffix) || upperKey == "KEY" || upperKey == "TOKEN" || upperKey == "SECRET" {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[i] = key + "=" + RedactedPlaceholder
		} else {
			sanitized[i] = key + "=" + SanitizeText(parts[1])
		}
	}

	return sanitized
}
