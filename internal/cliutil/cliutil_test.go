// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---- CleanText ----

func TestCleanText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"decodes numeric entity", "The Food Lab&#39;s Cookie", "The Food Lab's Cookie"},
		{"decodes named entity", "AT&amp;T", "AT&T"},
		{"trims whitespace", "  Chicken Tikka  ", "Chicken Tikka"},
		{"empty input", "", ""},
		{"plain passthrough", "Already clean.", "Already clean."},
		// Single-pass unescape contract: nested &amp;amp; decodes once to &amp;
		// but the inner &amp; stays encoded. If a caller needs repeated
		// unescaping they have a deeper upstream problem.
		{"single pass on nested entity", "&amp;amp;", "&amp;"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := CleanText(tc.in); got != tc.want {
				t.Errorf("CleanText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

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

// ---- ProbeReachable ----

// TestProbeReachable_200 asserts that a plain 2xx GET is classified
// reachable with the right code.
func TestProbeReachable_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	status, code, err := ProbeReachable(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != ReachabilityReachable {
		t.Errorf("status: want %q, got %q", ReachabilityReachable, status)
	}
	if code != 200 {
		t.Errorf("code: want 200, got %d", code)
	}
}

// TestProbeReachable_206_Reachable asserts hosts that honor Range
// (returning 206 Partial Content) are classified reachable.
func TestProbeReachable_206_Reachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer srv.Close()

	status, code, err := ProbeReachable(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != ReachabilityReachable {
		t.Errorf("status: want reachable, got %q", status)
	}
	if code != 206 {
		t.Errorf("code: want 206, got %d", code)
	}
}

// TestProbeReachable_416_Reachable asserts hosts that don't support
// Range (returning 416 Range Not Satisfiable) are still reachable —
// the headers came back, the host is up. This is the F4 motivating
// case: HEAD-then-GET probes incorrectly report unreachable here.
func TestProbeReachable_416_Reachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
	}))
	defer srv.Close()

	status, code, err := ProbeReachable(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != ReachabilityReachable {
		t.Errorf("status: want reachable (416 means headers came back), got %q", status)
	}
	if code != 416 {
		t.Errorf("code: want 416, got %d", code)
	}
}

// TestProbeReachable_403_Blocked asserts CDN bot screens (4xx other
// than 416) are classified blocked, not unreachable. The host is up
// and refusing this request — a doctor command should render WARN
// rather than FAIL.
func TestProbeReachable_403_Blocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	status, code, err := ProbeReachable(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != ReachabilityBlocked {
		t.Errorf("status: want blocked, got %q", status)
	}
	if code != 403 {
		t.Errorf("code: want 403, got %d", code)
	}
}

// TestProbeReachable_NetworkError_Unreachable asserts network-layer
// failures (DNS, connection refused, timeout) report unreachable with
// a non-nil err.
func TestProbeReachable_NetworkError_Unreachable(t *testing.T) {
	// Use a port that nothing is listening on.
	status, code, err := ProbeReachable(context.Background(), http.DefaultClient, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected non-nil err for unreachable host")
	}
	if status != ReachabilityUnreachable {
		t.Errorf("status: want unreachable, got %q", status)
	}
	if code != 0 {
		t.Errorf("code: want 0 (no response), got %d", code)
	}
}

// TestProbeReachable_NilClient_UsesDefault confirms the nil-client
// guard so doctor commands don't have to plumb an explicit *http.Client
// when default behavior is fine.
func TestProbeReachable_NilClient_UsesDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	status, _, err := ProbeReachable(context.Background(), nil, srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != ReachabilityReachable {
		t.Errorf("status: want reachable, got %q", status)
	}
}

// TestProbeReachable_NilClient_HasTimeout asserts the nil-client
// fallback uses a bounded-timeout client rather than http.DefaultClient
// (which has no timeout). Without this, a probe against a slow host
// could hang indefinitely. The test starts a server that hangs forever
// and relies on the default 10s timeout to bail out — capped to 12s
// total so a regression that drops the timeout would surface as a
// test failure rather than a hung test.
func TestProbeReachable_NilClient_HasTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skip slow timeout test in -short mode")
	}
	hang := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-hang // never returns until t.Cleanup closes it
	}))
	t.Cleanup(func() {
		close(hang)
		srv.Close()
	})

	done := make(chan struct{})
	var status string
	var probeErr error
	go func() {
		status, _, probeErr = ProbeReachable(context.Background(), nil, srv.URL)
		close(done)
	}()
	select {
	case <-done:
		// Probe returned within the bounded timeout — expected.
		if probeErr == nil {
			t.Fatalf("expected timeout err, got nil")
		}
		if status != ReachabilityUnreachable {
			t.Errorf("status: want unreachable on timeout, got %q", status)
		}
	case <-time.After(12 * time.Second):
		t.Fatalf("ProbeReachable hung past defaultProbeTimeout — nil-client fallback may be missing its bounded-timeout guard")
	}
}

// TestProbeReachable_SendsRangeHeader confirms the probe sends
// `Range: bytes=0-1023` so hosts that support Range bound the
// response body before we even read it.
func TestProbeReachable_SendsRangeHeader(t *testing.T) {
	var receivedRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRange = r.Header.Get("Range")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, _, err := ProbeReachable(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if receivedRange != "bytes=0-1023" {
		t.Errorf("Range header: want %q, got %q", "bytes=0-1023", receivedRange)
	}
}

// ---- AdaptiveLimiter / RateLimitError / RetryAfter / Backoff ----

func TestRateLimitError_ErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		err  *RateLimitError
		want string
	}{
		{
			name: "with retry-after and body",
			err:  &RateLimitError{URL: "https://api.example.com/x", RetryAfter: 5 * time.Second, Body: "slow down"},
			want: "rate limited: HTTP 429 for https://api.example.com/x; retry after 5s: slow down",
		},
		{
			name: "with retry-after no body",
			err:  &RateLimitError{URL: "https://api.example.com/x", RetryAfter: 2 * time.Second},
			want: "rate limited: HTTP 429 for https://api.example.com/x; retry after 2s",
		},
		{
			name: "no retry-after with body",
			err:  &RateLimitError{URL: "https://api.example.com/x", Body: "later"},
			want: "rate limited: HTTP 429 for https://api.example.com/x: later",
		},
		{
			name: "no retry-after no body",
			err:  &RateLimitError{URL: "https://api.example.com/x"},
			want: "rate limited: HTTP 429 for https://api.example.com/x",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRateLimitError_ErrorsAs(t *testing.T) {
	var err error = &RateLimitError{URL: "https://x", RetryAfter: time.Second}
	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *RateLimitError")
	}
	if target.URL != "https://x" {
		t.Errorf("target.URL = %q, want %q", target.URL, "https://x")
	}
}

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

func TestBackoff_DoublesPerAttempt(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
	}
	for _, tc := range cases {
		if got := Backoff(tc.attempt); got != tc.want {
			t.Errorf("Backoff(%d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestBackoff_CapsAtMax(t *testing.T) {
	if got := Backoff(20); got != MaxBackoff {
		t.Errorf("Backoff(20) = %v, want capped at %v", got, MaxBackoff)
	}
}

func TestBackoff_NegativeAttemptClampsToZero(t *testing.T) {
	if got := Backoff(-3); got != 1*time.Second {
		t.Errorf("Backoff(-3) = %v, want 1s (clamped to 0)", got)
	}
}
