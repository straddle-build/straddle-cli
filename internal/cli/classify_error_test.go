// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/client"
)

// TestClassifyAPIError pins the exit-code and hint-text mapping that
// classifyAPIError applies to *client.APIError values. The critical regression
// guard is the first group: an HTTP 400 whose JSON body carries a field named
// "key" or "token" must NOT be routed into the auth-error path (exit 4) — it
// must fall through to the generic API-error path (exit 5) so scripted
// consumers and users are not directed to re-export their API key for a
// validation error.
func TestClassifyAPIError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		err      *client.APIError
		wantCode int
		wantSub  string // substring that must appear in the classified error; "" = none
		notSub   string // substring that must NOT appear; "" = no check
	}{
		// --- HTTP 400: auth-shaped bodies → exit 4 with auth hint ---
		{
			name:     "400 missing api_key",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"missing api_key"}}`},
			wantCode: 4,
			wantSub:  "auth is missing or invalid",
		},
		{
			name:     "400 invalid api key",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"Invalid API key"}}`},
			wantCode: 4,
			wantSub:  "auth is missing or invalid",
		},
		{
			name:     "400 unauthorized",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"Unauthorized"}}`},
			wantCode: 4,
			wantSub:  "auth is missing or invalid",
		},
		{
			name:     "400 forbidden",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"Forbidden"}}`},
			wantCode: 4,
			wantSub:  "auth is missing or invalid",
		},
		{
			name:     "400 invalid credentials",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"invalid credentials"}}`},
			wantCode: 4,
			wantSub:  "auth is missing or invalid",
		},
		{
			name:     "400 expired token",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"expired token"}}`},
			wantCode: 4,
			wantSub:  "auth is missing or invalid",
		},

		// --- HTTP 400: non-auth validation bodies → exit 5, NO auth hint ---
		// The original bug: \bkey\b matched the JSON field name "key".
		{
			name:     "400 json key field",
			err:      &client.APIError{Method: "GET", Path: "/v1/checkout_sessions", StatusCode: 400, Body: `{"error":{"code":"validation_error","key":"amount"}}`},
			wantCode: 5,
			notSub:   "auth is missing or invalid",
		},
		// "primary key" is a database concept, not auth.
		{
			name:     "400 primary key constraint",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"error":{"message":"primary key constraint violated"}}`},
			wantCode: 5,
			notSub:   "auth is missing or invalid",
		},
		// JSON field named "token" — \btoken\b matched it before the fix.
		{
			name:     "400 json token field",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `{"token":"abc123"}`},
			wantCode: 5,
			notSub:   "auth is missing or invalid",
		},
		// Idempotency-Key header prose — contains "Key" but is about idempotency.
		{
			name:     "400 idempotency key",
			err:      &client.APIError{Method: "POST", Path: "/v1/charges", StatusCode: 400, Body: `The Idempotency-Key header is invalid. It must be between 10 and 40 characters long.`},
			wantCode: 5,
			notSub:   "auth is missing or invalid",
		},
		// No auth keywords at all.
		{
			name:     "400 malformed page number",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 400, Body: `malformed page number`},
			wantCode: 5,
			notSub:   "auth is missing or invalid",
		},
		{
			name:     "400 file signature",
			err:      &client.APIError{Method: "POST", Path: "/v1/charges/upload-authorization", StatusCode: 400, Body: `{"error":{"title":"File signature does not match its extension"}}`},
			wantCode: 5,
			notSub:   "auth is missing or invalid",
		},

		// --- Other HTTP status codes: verify no regression ---
		{
			name:     "401 check token hint",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 401, Body: `{"error":{"message":"unauthorized"}}`},
			wantCode: 4,
			wantSub:  "check your token",
		},
		{
			name:     "403 permission denied hint",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 403, Body: `{"error":{"message":"forbidden"}}`},
			wantCode: 4,
			wantSub:  "permission denied",
		},
		{
			name:     "404 not found hint",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges/abc", StatusCode: 404, Body: `not found`},
			wantCode: 3,
			wantSub:  "resource not found",
		},
		{
			name:     "429 rate limit",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 429, Body: `rate limited`},
			wantCode: 7,
		},
		{
			name:     "500 server error",
			err:      &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 500, Body: `internal server error`},
			wantCode: 5,
		},
		{
			name:     "409 conflict",
			err:      &client.APIError{Method: "POST", Path: "/v1/charges", StatusCode: 409, Body: `conflict`},
			wantCode: 5,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			classified := classifyAPIError(tc.err, nil)
			if got := ExitCode(classified); got != tc.wantCode {
				t.Fatalf("ExitCode = %d, want %d; err=%v", got, tc.wantCode, classified)
			}
			msg := ""
			if classified != nil {
				msg = classified.Error()
			}
			if tc.wantSub != "" && !strings.Contains(msg, tc.wantSub) {
				t.Fatalf("error missing %q; got %q", tc.wantSub, msg)
			}
			if tc.notSub != "" && strings.Contains(msg, tc.notSub) {
				t.Fatalf("error unexpectedly contains %q; got %q", tc.notSub, msg)
			}
		})
	}
}
