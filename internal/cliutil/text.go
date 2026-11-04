// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

// Package cliutil contains shared helpers used across the Straddle CLI.
// Helpers live in their own package (not in package cli) to avoid symbol
// collisions with agent-authored commands in package cli. Callers import as
// `cliutil` and invoke `cliutil.RetryAfter(...)`,
// `cliutil.RedactCredentials(...)`, etc.
package cliutil

import (
	"regexp"
	"strings"
)

// LooksLikeAuthError checks if an error message body contains auth-related keywords.
func LooksLikeAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	patterns := []string{
		`\bkey\b`,
		`\btoken\b`,
		`\bunauthorized\b`,
		`\bapi_key\b`,
		`missing.{0,20}key`,
		`required.{0,20}key`,
		`\bforbidden\b`,
		`\bauthenticat`,
		`\bcredential`,
	}
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, lower); matched {
			return true
		}
	}
	return false
}

// credPatterns matches credential-shaped substrings (API keys, bearer
// tokens, key=... query params) for redaction from user-visible output.
var credPatterns = regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{8,}|sk_live_[a-zA-Z0-9]+|Bearer\s+[a-zA-Z0-9._+/=-]+|key=(?:[a-zA-Z0-9._+/=-]|%[0-9a-f]{2})+)`)

// RedactCredentials strips credential-shaped strings from s without
// truncating it. Applied to API error bodies at the source so a hostile
// or misconfigured server cannot reflect a credential back through CLI
// error output.
func RedactCredentials(s string) string {
	return credPatterns.ReplaceAllString(s, "[REDACTED]")
}

// SanitizeErrorBody truncates and strips credential-shaped strings from error output.
func SanitizeErrorBody(msg string) string {
	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}
	return RedactCredentials(msg)
}
