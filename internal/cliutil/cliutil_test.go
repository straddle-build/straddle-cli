// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestParseStoredTime(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want time.Time
	}{
		{
			name: "rfc3339 nano",
			in:   "2026-04-21T09:02:49.123456789-07:00",
			want: time.Date(2026, 4, 21, 9, 2, 49, 123456789, time.FixedZone("", -7*60*60)),
		},
		{
			name: "modernc go string",
			in:   "2026-04-21 09:02:49.123456789 -0700 PDT",
			want: time.Date(2026, 4, 21, 9, 2, 49, 123456789, time.FixedZone("PDT", -7*60*60)),
		},
		{
			name: "blank",
			in:   "",
			want: time.Time{},
		},
		{
			name: "invalid",
			in:   "not a time",
			want: time.Time{},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ParseStoredTime(tc.in)
			if !got.Equal(tc.want) {
				t.Fatalf("ParseStoredTime(%q) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestAuthErrorHelpers(t *testing.T) {
	if !LooksLikeAuthError("HTTP 400: missing api_key") {
		t.Fatal("expected missing api_key to look like an auth error")
	}
	if LooksLikeAuthError("HTTP 400: malformed page number") {
		t.Fatal("unexpected auth classification for non-auth message")
	}

	got := SanitizeErrorBody("token sk-abcdefghi Bearer abc.def key=secretvalue")
	if got != "token [REDACTED] [REDACTED] [REDACTED]" {
		t.Fatalf("SanitizeErrorBody redaction = %q", got)
	}
}

// TestLooksLikeAuthError pins the heuristic that classifies HTTP 400 responses
// as auth-related. The patterns must catch genuinely auth-shaped messages
// (missing/invalid api key, expired/revoked token, unauthorized, forbidden,
// credentials) while rejecting ordinary validation errors whose JSON bodies
// happen to carry a field named "key" or "token" — the regression that motivated
// tightening the bare \bkey\b / \btoken\b patterns.
func TestLooksLikeAuthError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		msg  string
		want bool
	}{
		// --- Positive: genuinely auth-shaped 400 bodies ---
		{"missing api_key", "GET /v1/charges returned HTTP 400: missing api_key", true},
		{"missing api key space", "GET /v1/charges returned HTTP 400: Missing API key", true},
		{"api-key hyphen", "GET /v1/charges returned HTTP 400: invalid api-key", true},
		{"apikey no separator", "GET /v1/charges returned HTTP 400: invalid apikey", true},
		{"api_key underscore", "GET /v1/charges returned HTTP 400: invalid api_key", true},
		{"api key is required", "GET /v1/charges returned HTTP 400: API key is required", true},
		{"missing key", "GET /v1/charges returned HTTP 400: missing key", true},
		{"required key", "GET /v1/charges returned HTTP 400: required key", true},
		{"invalid key", "GET /v1/charges returned HTTP 400: invalid key", true},
		{"unauthorized", "GET /v1/charges returned HTTP 400: Unauthorized", true},
		{"forbidden", "GET /v1/charges returned HTTP 400: Forbidden", true},
		{"authentication required", "GET /v1/charges returned HTTP 400: Authentication required", true},
		{"credential", "GET /v1/charges returned HTTP 400: invalid credential", true},
		{"credentials", "GET /v1/charges returned HTTP 400: invalid credentials", true},
		{"missing access token", "GET /v1/charges returned HTTP 400: missing access token", true},
		{"access_token underscore", "GET /v1/charges returned HTTP 400: missing access_token", true},
		{"invalid auth token", "GET /v1/charges returned HTTP 400: invalid auth token", true},
		{"bearer token", "GET /v1/charges returned HTTP 400: invalid bearer token", true},
		{"missing token", "GET /v1/charges returned HTTP 400: missing token", true},
		{"invalid token", "GET /v1/charges returned HTTP 400: invalid token", true},
		{"expired token", "GET /v1/charges returned HTTP 400: expired token", true},
		{"revoked token", "GET /v1/charges returned HTTP 400: revoked token", true},
		{"required token", "GET /v1/charges returned HTTP 400: required token", true},

		// --- Negative: non-auth 400 bodies that must NOT be classified as auth ---
		// The original bug: JSON field named "key" matched \bkey\b because quotes
		// are word boundaries.
		{"json key field", `GET /v1/checkout_sessions returned HTTP 400: {"error":{"code":"validation_error","key":"amount"}}`, false},
		// Prose containing "primary key" — a database concept, not auth.
		{"primary key constraint", `GET /v1/charges returned HTTP 400: {"error":{"message":"primary key constraint violated"}}`, false},
		// JSON field named "token" — \btoken\b matched it before the fix.
		{"json token field", `GET /v1/charges returned HTTP 400: {"token":"abc123"}`, false},
		// Snake_case field token_expiry — \btoken\b missed this even before the
		// fix (underscore is a word char), but pin it to guard against future
		// pattern loosening.
		{"token_expiry field", `GET /v1/charges returned HTTP 400: {"error":{"field":"token_expiry","reason":"expired"}}`, false},
		// Idempotency-Key header prose — contains "Key" but is about idempotency.
		{"idempotency key", `GET /v1/charges returned HTTP 400: The Idempotency-Key header is invalid. It must be between 10 and 40 characters long.`, false},
		// No auth keywords at all.
		{"malformed page number", "GET /v1/charges returned HTTP 400: malformed page number", false},
		{"file signature", "GET /v1/charges returned HTTP 400: File signature does not match its extension", false},
		// "invalid" without key/token nearby.
		{"invalid amount", `GET /v1/charges returned HTTP 400: {"error":{"message":"invalid amount"}}`, false},
		// "expired" without token nearby.
		{"expired session", `GET /v1/charges returned HTTP 400: {"error":{"message":"session expired"}}`, false},
		// "key" and "token" as values, not field names or auth terms.
		{"key as value", `GET /v1/charges returned HTTP 400: {"error":{"sort":"key","direction":"desc"}}`, false},
		{"token as value", `GET /v1/charges returned HTTP 400: {"error":{"type":"token","position":3}}`, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := LooksLikeAuthError(tc.msg); got != tc.want {
				t.Fatalf("LooksLikeAuthError(%q) = %v, want %v", tc.msg, got, tc.want)
			}
		})
	}
}

// ---- AdaptiveLimiter / RetryAfter ----

func TestRetryAfter_Seconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "10")
	if got := RetryAfter(resp); got != 10*time.Second {
		t.Errorf("RetryAfter(10) = %v, want 10s", got)
	}
}

func TestRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(7 * time.Second).UTC().Format(http.TimeFormat)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", future)
	got := RetryAfter(resp)
	if got < 5*time.Second || got > 8*time.Second {
		t.Errorf("RetryAfter(http-date 7s ahead) = %v, want ~7s", got)
	}
}

func TestRetryAfter_EpochSeconds(t *testing.T) {
	future := time.Now().Add(7 * time.Second)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", fmt.Sprint(future.Unix()))
	got := RetryAfter(resp)
	if got < 5*time.Second || got > 8*time.Second {
		t.Errorf("RetryAfter(epoch seconds 7s ahead) = %v, want ~7s", got)
	}
}

func TestRetryAfter_EpochMilliseconds(t *testing.T) {
	future := time.Now().Add(7 * time.Second)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", fmt.Sprint(future.UnixMilli()))
	got := RetryAfter(resp)
	if got < 5*time.Second || got > 8*time.Second {
		t.Errorf("RetryAfter(epoch milliseconds 7s ahead) = %v, want ~7s", got)
	}
}

func TestRetryAfter_Cap(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "600")
	if got := RetryAfter(resp); got != MaxRetryWait {
		t.Errorf("RetryAfter(600) = %v, want capped at %v", got, MaxRetryWait)
	}
}

func TestRetryAfter_Missing(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	if got := RetryAfter(resp); got != 5*time.Second {
		t.Errorf("RetryAfter(missing) = %v, want 5s default", got)
	}
}

func TestRetryAfter_MalformedFallsBackToDefault(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "not-a-number")
	if got := RetryAfter(resp); got != 5*time.Second {
		t.Errorf("RetryAfter(garbage) = %v, want 5s default", got)
	}
}

func TestRetryAfter_NilResp(t *testing.T) {
	if got := RetryAfter(nil); got != 5*time.Second {
		t.Errorf("RetryAfter(nil) = %v, want 5s default", got)
	}
}
