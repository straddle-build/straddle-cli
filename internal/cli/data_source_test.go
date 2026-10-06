// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/straddle-build/straddle-cli/internal/client"
	"github.com/straddle-build/straddle-cli/internal/store"
)

// No t.Parallel(): captureStderr swaps the process-global os.Stderr.
func TestWriteThroughCacheDetailEnvelopeDiagnostics(t *testing.T) {
	const meta = `{"api_request_id":"a1b2c3d4-0000-4000-8000-000000000001","api_request_timestamp":"2026-09-26T12:00:00Z"}`
	cases := []struct {
		name        string
		resource    string
		wantRows    int
		wantWarning bool
	}{
		{name: "valid detail stores inner resource silently", resource: `{"id":"0f5b2c4e-9a1d-4c3b-8e2f-1a2b3c4d5e6f","status":"created","amount":1250}`, wantRows: 1},
		{name: "detail without resource ID stores nothing and warns", resource: `{"status":"created","amount":1250}`, wantWarning: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			envelope := `{"meta":` + meta + `,"response_type":"object","data":` + tc.resource + `}`
			stderr := captureStderr(t, func() {
				writeThroughCache(context.Background(), "charges", json.RawMessage(envelope))
			})

			db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"), testStoreScope(t))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			rows, err := db.Count("charges")
			if err != nil {
				t.Fatal(err)
			}
			if rows != tc.wantRows {
				t.Fatalf("stored charges = %d, want %d", rows, tc.wantRows)
			}
			if tc.wantRows == 1 {
				got, err := db.Get("charges", "0f5b2c4e-9a1d-4c3b-8e2f-1a2b3c4d5e6f")
				if err != nil {
					t.Fatalf("charge not stored under its resource ID: %v", err)
				}
				var gotResource, wantResource map[string]any
				if err := json.Unmarshal(got, &gotResource); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(tc.resource), &wantResource); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(gotResource, wantResource) {
					t.Fatalf("stored charge = %s, want inner resource %s", got, tc.resource)
				}
			}
			warned := strings.Contains(stderr, "warning") && strings.Contains(stderr, "charges")
			if warned != tc.wantWarning {
				t.Fatalf("charges warning emitted = %t, want %t; stderr=%q", warned, tc.wantWarning, stderr)
			}
			if !tc.wantWarning && stderr != "" {
				t.Fatalf("valid detail wrote diagnostics: %q", stderr)
			}
		})
	}
}

func TestWriteThroughCacheSkipsSensitiveObjectEnvelope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	payload := json.RawMessage(`{"meta":{},"response_type":"object","data":{"id":"secret-123","ssn":"masked"}}`)
	writeThroughCache(context.Background(), "unmask", payload)
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"), testStoreScope(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Get("unmask", "secret-123"); err == nil {
		t.Fatal("sensitive detail unexpectedly cached")
	}
}

func TestWriteThroughCachePreservesBareObjectWithDataField(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeThroughCache(context.Background(), "charges", json.RawMessage(`{"id":"outer","data":{"id":"inner"},"status":"pending"}`))
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"), testStoreScope(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Get("charges", "outer"); err != nil {
		t.Fatalf("outer resource not cached: %v", err)
	}
	if _, err := db.Get("charges", "inner"); err == nil {
		t.Fatal("inner collision object was cached")
	}
}

func TestWriteThroughCacheSkipsRevealEnvelope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	payload := json.RawMessage(`{"meta":{},"response_type":"object","data":{"id":"secret-456"}}`)
	writeThroughCache(context.Background(), "reveal", payload)
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"), testStoreScope(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Get("reveal", "secret-456"); err == nil {
		t.Fatal("revealed detail unexpectedly cached")
	}
}

func TestResolveReadRequestWritesDetailForOfflineLookup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	flags := &rootFlags{dataSource: "auto"}
	live := func() (json.RawMessage, error) {
		return json.RawMessage(`{"meta":{},"response_type":"object","data":{"id":"ch_auto","status":"pending"}}`), nil
	}
	if _, _, err := resolveReadRequest(context.Background(), flags, "charges", false, "/v1/charges/ch_auto", nil, live); err != nil {
		t.Fatal(err)
	}
	flags.dataSource = "local"
	got, _, err := resolveReadRequest(context.Background(), flags, "charges", false, "/v1/charges/ch_auto", nil, func() (json.RawMessage, error) { t.Fatal("local read called live function"); return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"id":"ch_auto","status":"pending"}` {
		t.Fatalf("offline result = %s", got)
	}
}

// httpClientTimeoutError reproduces the production http.Client.Timeout
// failure shape: *http.timeoutError ("context deadline exceeded
// (Client.Timeout exceeded while awaiting headers)") wrapped in a
// *url.Error. It is produced by pointing a short-timeout client at a
// server that accepts the connection but never responds — the real-world
// "API hung" outage. *http.timeoutError is an unexported stdlib type and
// cannot be constructed directly, so a live round-trip is the faithful
// way to obtain it. The handler unblocks on request-context cancellation
// so the test server shuts down cleanly without leaking a goroutine.
func httpClientTimeoutError(t *testing.T) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	c := &http.Client{Timeout: 50 * time.Millisecond}
	_, err := c.Get(srv.URL)
	if err == nil {
		t.Fatal("expected http.Client.Timeout error, got nil")
	}
	return err
}

// dialRefusedError reproduces a connection-refused failure (*url.Error
// wrapping *net.OpError with "connection refused") by dialing an
// httptest server port that has already been closed. The dial returns
// immediately, so this is fast and deterministic on loopback.
func dialRefusedError(t *testing.T) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.Listener.Addr().String()
	srv.Close()
	_, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://" + addr + "/")
	if err == nil {
		t.Fatal("expected connection refused error, got nil")
	}
	return err
}

// TestIsNetworkError pins the offline-fallback classification gate.
// The	Client.Timeout path (the bug: *http.timeoutError / "context
// deadline exceeded") must be recognized as a network error so
// resolveReadRequest/resolvePaginatedReadRequest fall back to the local
// cache. HTTP 4xx/5xx APIError and context.Canceled must NOT be
// classified as network errors.
func TestIsNetworkError(t *testing.T) {
	// Real *http.timeoutError (read-hang) wrapped in *url.Error.
	clientTimeoutURL := httpClientTimeoutError(t)
	var clientTimeoutBare error
	var ue *url.Error
	if errors.As(clientTimeoutURL, &ue) {
		clientTimeoutBare = ue.Err
	}
	// Real connection-refused *url.Error wrapping *net.OpError.
	dialRefused := dialRefusedError(t)

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "unrelated error", err: errors.New("something else entirely"), want: false},
		{name: "HTTP 4xx APIError", err: &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 404, Body: "not found"}, want: false},
		{name: "HTTP 5xx APIError", err: &client.APIError{Method: "GET", Path: "/v1/charges", StatusCode: 500, Body: "boom"}, want: false},
		{name: "context.Canceled is NOT a network error (regression guard)", err: context.Canceled, want: false},
		{name: "context.DeadlineExceeded directly", err: context.DeadlineExceeded, want: true},
		{name: "connection refused string", err: errors.New("dial tcp 127.0.0.1:80: connect: connection refused"), want: true},
		{name: "no such host string", err: errors.New("dial tcp: lookup nonexistent.invalid: no such host"), want: true},
		{name: "network unreachable string", err: errors.New("dial tcp: network is unreachable"), want: true},
		{name: "i/o timeout string (dialer)", err: errors.New("dial tcp 192.0.2.1:80: i/o timeout"), want: true},
		{name: "TLS handshake timeout string", err: errors.New("net/http: TLS handshake timeout"), want: true},
		{name: "constructed *net.OpError (dial)", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}, want: true},
		{name: "constructed *net.DNSError", err: &net.DNSError{Name: "nonexistent.invalid", Err: "no such host", IsNotFound: true}, want: true},
		{name: "*url.Error wrapping *net.OpError (real dial refused)", err: dialRefused, want: true},
		{name: "*url.Error wrapping *http.timeoutError (real Client.Timeout)", err: clientTimeoutURL, want: true},
		{name: "bare *http.timeoutError unwrapped from *url.Error", err: clientTimeoutBare, want: true},
		{name: "*url.Error wrapping context.DeadlineExceeded (synthetic Client.Timeout)", err: &url.Error{Op: "Get", URL: "https://api.example.com/v1/charges", Err: context.DeadlineExceeded}, want: true},
		{name: "*url.Error wrapping constructed *net.OpError i/o timeout", err: &url.Error{Op: "Get", URL: "http://192.0.2.1/", Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("dial tcp 192.0.2.1:80: i/o timeout")}}, want: true},
		{name: "retry-loop wrap of *url.Error Client.Timeout", err: fmt.Errorf("GET /v1/charges: %w", clientTimeoutURL), want: true},
		{name: "retry-loop wrap of *url.Error dial refused", err: fmt.Errorf("GET /v1/charges: %w", dialRefused), want: true},
		{name: "retry-loop wrap of *url.Error/DeadlineExceeded (synthetic)", err: fmt.Errorf("GET /v1/charges: %w", &url.Error{Op: "Get", URL: "https://api.example.com/v1/charges", Err: context.DeadlineExceeded}), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNetworkError(tc.err); got != tc.want {
				t.Fatalf("isNetworkError(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

// TestResolveReadRequestFallsBackOnClientTimeout verifies the bug fix
// end-to-end: when the live API read fails with an http.Client.Timeout
// (server hung / dial-blackhole at default --timeout), the auto data
// source must fall back to the local cache instead of returning the raw
// timeout error.
func TestResolveReadRequestFallsBackOnClientTimeout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Prime the local cache exactly as a prior successful live read would.
	writeThroughCache(context.Background(), "charges", json.RawMessage(`{"meta":{},"response_type":"object","data":{"id":"ch_timeout","status":"pending"}}`))

	flags := &rootFlags{dataSource: "auto"}
	timeoutErr := httpClientTimeoutError(t)
	live := func() (json.RawMessage, error) { return nil, timeoutErr }

	got, prov, err := resolveReadRequest(context.Background(), flags, "charges", false, "/v1/charges/ch_timeout", nil, live)
	if err != nil {
		t.Fatalf("expected offline fallback on Client.Timeout, got error: %v", err)
	}
	if string(got) != `{"id":"ch_timeout","status":"pending"}` {
		t.Fatalf("offline result = %s, want {\"id\":\"ch_timeout\",\"status\":\"pending\"}", got)
	}
	if prov.Source != "local" {
		t.Fatalf("provenance source = %q, want %q", prov.Source, "local")
	}
	if prov.Reason != "api_unreachable" {
		t.Fatalf("provenance reason = %q, want %q", prov.Reason, "api_unreachable")
	}
}

// TestResolvePaginatedReadRequestFallsBackOnClientTimeout mirrors the
// detail test for the paginated read path, which also gates fallback on
// isNetworkError.
func TestResolvePaginatedReadRequestFallsBackOnClientTimeout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeThroughCache(context.Background(), "charges", json.RawMessage(`[{"id":"ch_p1"},{"id":"ch_p2"}]`))

	flags := &rootFlags{dataSource: "auto"}
	timeoutErr := httpClientTimeoutError(t)
	live := func() (json.RawMessage, error) { return nil, timeoutErr }

	got, prov, err := resolvePaginatedReadRequest(context.Background(), flags, "charges", "/v1/charges", nil, live)
	if err != nil {
		t.Fatalf("expected offline fallback on Client.Timeout, got error: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(got, &items); err != nil {
		t.Fatalf("unmarshaling local fallback data: %v: %s", err, got)
	}
	if len(items) != 2 {
		t.Fatalf("local fallback returned %d items, want 2", len(items))
	}
	gotIDs := map[string]bool{}
	for _, it := range items {
		if id, ok := it["id"].(string); ok {
			gotIDs[id] = true
		}
	}
	for _, want := range []string{"ch_p1", "ch_p2"} {
		if !gotIDs[want] {
			t.Fatalf("local fallback missing id %q; got %v", want, gotIDs)
		}
	}
	if prov.Source != "local" {
		t.Fatalf("provenance source = %q, want %q", prov.Source, "local")
	}
	if prov.Reason != "api_unreachable" {
		t.Fatalf("provenance reason = %q, want %q", prov.Reason, "api_unreachable")
	}
}

// TestResolveReadRequestReturnsTimeoutErrorWhenNoLocalData ensures the
// fallback surfaces the documented "API unreachable and no local data"
// error (wrapping the original timeout) when the cache is empty, so the
// timeout case behaves like every other network error rather than
// silently returning empty data.
func TestResolveReadRequestReturnsTimeoutErrorWhenNoLocalData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home) // no sync run -> no local store

	flags := &rootFlags{dataSource: "auto"}
	timeoutErr := httpClientTimeoutError(t)
	live := func() (json.RawMessage, error) { return nil, timeoutErr }

	_, _, err := resolveReadRequest(context.Background(), flags, "charges", false, "/v1/charges/ch_missing", nil, live)
	if err == nil {
		t.Fatal("expected error when API unreachable and no local data, got nil")
	}
	if !strings.Contains(err.Error(), "API unreachable") {
		t.Fatalf("error missing \"API unreachable\" guidance: %v", err)
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("error missing original timeout cause: %v", err)
	}
}
