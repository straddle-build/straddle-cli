// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

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
