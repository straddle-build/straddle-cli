// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/straddle-build/straddle-cli/internal/config"
)

// retryRecorder answers every request with the same failure and records the
// Idempotency-Key each attempt carried.
type retryRecorder struct {
	keys []string
	// status is the HTTP status to answer with; zero answers with a
	// network error instead.
	status int
}

func (r *retryRecorder) client(t *testing.T, cfg *config.Config) *Client {
	t.Helper()
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.BaseURL = "https://api.example.com"
	c := New(cfg, time.Second, 0)
	c.NoCache = true
	c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		r.keys = append(r.keys, req.Header.Get("Idempotency-Key"))
		if r.status == 0 {
			return nil, errors.New("connection reset by peer")
		}
		return &http.Response{
			StatusCode: r.status,
			Body:       io.NopCloser(strings.NewReader(`{"error":"failed"}`)),
		}, nil
	})
	return c
}

// assertOneKey fails unless every attempt carried the same non-empty key.
func assertOneKey(t *testing.T, keys []string) string {
	t.Helper()
	if len(keys) == 0 {
		t.Fatal("no request was sent")
	}
	for i, key := range keys {
		if key == "" || key != keys[0] {
			t.Fatalf("attempt %d Idempotency-Key = %q, want every attempt to send %q", i+1, key, keys[0])
		}
	}
	return keys[0]
}

func TestClient_RetriedWriteReusesOneGeneratedIdempotencyKey(t *testing.T) {
	tests := []struct {
		name   string
		method string
		status int
	}{
		{name: "POST server error", method: http.MethodPost, status: http.StatusServiceUnavailable},
		{name: "POST network error", method: http.MethodPost},
		{name: "POST rate limited", method: http.MethodPost, status: http.StatusTooManyRequests},
		{name: "PUT server error", method: http.MethodPut, status: http.StatusBadGateway},
		{name: "PATCH server error", method: http.MethodPatch, status: http.StatusInternalServerError},
		{name: "DELETE server error", method: http.MethodDelete, status: http.StatusGatewayTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				rec := &retryRecorder{status: tt.status}
				c := rec.client(t, nil)

				if _, _, err := c.do(tt.method, "/v1/charges", nil, map[string]any{"amount": 100}, nil); err == nil {
					t.Fatal("do() error = nil, want the final failure")
				}

				if len(rec.keys) != 4 {
					t.Fatalf("attempts = %d, want 4", len(rec.keys))
				}
				key := assertOneKey(t, rec.keys)
				if len(key) < 10 || len(key) > 40 {
					t.Fatalf("generated Idempotency-Key %q has %d characters, want 10 to 40", key, len(key))
				}
			})
		})
	}
}

func TestClient_EachWriteGetsItsOwnIdempotencyKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &retryRecorder{status: http.StatusBadRequest}
		c := rec.client(t, nil)

		_, _, _ = c.Post("/v1/charges", map[string]any{"amount": 100})
		_, _, _ = c.Post("/v1/charges", map[string]any{"amount": 100})

		if len(rec.keys) != 2 {
			t.Fatalf("attempts = %d, want 2 (a 4xx is not retried)", len(rec.keys))
		}
		if rec.keys[0] == "" || rec.keys[0] == rec.keys[1] {
			t.Fatalf("keys = %q, want two distinct non-empty keys for two separate writes", rec.keys)
		}
	})
}

func TestClient_RetriedWriteKeepsCallerIdempotencyKey(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		headers map[string]string
	}{
		{name: "command flag", headers: map[string]string{"Idempotency-Key": "caller-key-0001"}},
		{name: "lowercase command header", headers: map[string]string{"idempotency-key": "caller-key-0001"}},
		{name: "config headers", cfg: &config.Config{Headers: map[string]string{"Idempotency-Key": "caller-key-0001"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				rec := &retryRecorder{status: http.StatusServiceUnavailable}
				c := rec.client(t, tt.cfg)

				_, _, _ = c.PostWithHeaders("/v1/charges", map[string]any{"amount": 100}, tt.headers)

				if len(rec.keys) != 4 {
					t.Fatalf("attempts = %d, want 4", len(rec.keys))
				}
				if key := assertOneKey(t, rec.keys); key != "caller-key-0001" {
					t.Fatalf("Idempotency-Key = %q, want the caller's key", key)
				}
			})
		})
	}
}

func TestClient_WriteDoesNotMutateCallerHeaders(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &retryRecorder{status: http.StatusBadRequest}
		c := rec.client(t, nil)
		headers := map[string]string{"Correlation-Id": "trace-1"}

		_, _, _ = c.PostWithHeaders("/v1/charges", map[string]any{"amount": 100}, headers)

		if _, ok := headers["Idempotency-Key"]; ok || len(headers) != 1 {
			t.Fatalf("caller headers = %v, want only Correlation-Id", headers)
		}
		if rec.keys[0] == "" {
			t.Fatal("write sent no Idempotency-Key")
		}
	})
}

func TestClient_ReadsRetryWithoutIdempotencyKey(t *testing.T) {
	tests := []struct {
		name   string
		status int
		send   func(*Client) error
	}{
		{
			name:   "GET server error",
			status: http.StatusServiceUnavailable,
			send: func(c *Client) error {
				_, err := c.Get("/v1/charges/ch_1", nil)
				return err
			},
		},
		{
			name: "GET network error",
			send: func(c *Client) error {
				_, err := c.Get("/v1/charges/ch_1", nil)
				return err
			},
		},
		{
			name:   "read-only POST server error",
			status: http.StatusServiceUnavailable,
			send: func(c *Client) error {
				_, _, err := c.PostQueryWithParams("/v1/charges/search", nil, map[string]any{})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				rec := &retryRecorder{status: tt.status}
				c := rec.client(t, nil)

				if err := tt.send(c); err == nil {
					t.Fatal("error = nil, want the final failure")
				}

				if len(rec.keys) != 4 {
					t.Fatalf("attempts = %d, want 4", len(rec.keys))
				}
				for i, key := range rec.keys {
					if key != "" {
						t.Fatalf("attempt %d Idempotency-Key = %q, want none on a read", i+1, key)
					}
				}
			})
		})
	}
}
