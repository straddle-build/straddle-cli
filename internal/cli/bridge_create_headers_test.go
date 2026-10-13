// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type capturedBridgeRequest struct {
	method  string
	path    string
	headers http.Header
}

func bridgeCreateTanRequiredArgs() []string {
	return []string{
		"--account-type=checking",
		"--customer-id=" + goldenUUID,
		"--routing-number=110000000",
		"--tan=tan-idem-token",
	}
}

func bridgeCreateSpeedchexRequiredArgs() []string {
	return []string{
		"--customer-id=" + goldenUUID,
		"--speedchex-token=speedchex-idem-token",
	}
}

func TestBridgeCreateTanHeadersReachServer(t *testing.T) {
	runBridgeCreateHeadersReachServer(t, "tan", "/v1/bridge/tan", bridgeCreateTanRequiredArgs())
}

func TestBridgeCreateSpeedchexHeadersReachServer(t *testing.T) {
	runBridgeCreateHeadersReachServer(t, "speedchex", "/v1/bridge/speedchex", bridgeCreateSpeedchexRequiredArgs())
}

func runBridgeCreateHeadersReachServer(t *testing.T, sub, wantPath string, requiredArgs []string) {
	var (
		mu       sync.Mutex
		gotReq   capturedBridgeRequest
		requests int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		gotReq = capturedBridgeRequest{method: r.Method, path: r.URL.Path, headers: r.Header.Clone()}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"created"}}`))
	}))
	t.Cleanup(server.Close)
	isolateSurfaceConfig(t, server.URL)

	headerArgs := []string{
		"--correlation-id=cid-bridge-" + sub,
		"--idempotency-key=idem-bridge-" + sub,
		"--request-id=rid-bridge-" + sub,
	}
	args := append([]string{"--json", "--no-cache", "bridge", "create-" + sub}, requiredArgs...)
	args = append(args, headerArgs...)

	if _, _, err := runRootForAPITest(t, args, ""); err != nil {
		t.Fatalf("bridge create-%s returned error: %v", sub, err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("sent %d requests, want 1", requests)
	}
	if gotReq.method != http.MethodPost || gotReq.path != wantPath {
		t.Fatalf("request = %s %s, want POST %s", gotReq.method, gotReq.path, wantPath)
	}
	for _, tc := range []struct{ header, want string }{
		{"Correlation-Id", "cid-bridge-" + sub},
		{"Idempotency-Key", "idem-bridge-" + sub},
		{"Request-Id", "rid-bridge-" + sub},
	} {
		if got := gotReq.headers.Get(tc.header); got != tc.want {
			t.Errorf("%s header = %q, want %q", tc.header, got, tc.want)
		}
	}
}

func TestBridgeCreateHeadersOmittedWhenFlagsAbsent(t *testing.T) {
	for _, tc := range []struct {
		sub          string
		requiredArgs []string
	}{
		{"tan", bridgeCreateTanRequiredArgs()},
		{"speedchex", bridgeCreateSpeedchexRequiredArgs()},
	} {
		t.Run(tc.sub, func(t *testing.T) {
			var requests int
			var headers http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				headers = r.Header.Clone()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"id":"created"}}`))
			}))
			t.Cleanup(server.Close)
			isolateSurfaceConfig(t, server.URL)

			args := append([]string{"--json", "--no-cache", "bridge", "create-" + tc.sub}, tc.requiredArgs...)
			if _, _, err := runRootForAPITest(t, args, ""); err != nil {
				t.Fatalf("bridge create-%s returned error: %v", tc.sub, err)
			}
			if requests != 1 {
				t.Fatalf("sent %d requests, want 1", requests)
			}
			for _, h := range []string{"Correlation-Id", "Idempotency-Key", "Request-Id"} {
				if v := headers.Get(h); v != "" {
					t.Errorf("%s header = %q, want absent when flag not set", h, v)
				}
			}
		})
	}
}

func TestBridgeCreateHeadersDryRunPreview(t *testing.T) {
	for _, tc := range []struct {
		sub          string
		requiredArgs []string
	}{
		{"tan", bridgeCreateTanRequiredArgs()},
		{"speedchex", bridgeCreateSpeedchexRequiredArgs()},
	} {
		t.Run(tc.sub, func(t *testing.T) {
			isolateAPIConfig(t)
			t.Setenv("STRADDLE_API_KEY", "test_key")
			t.Setenv("STRADDLE_BASE_URL", "http://127.0.0.1:1")

			args := append([]string{"--dry-run", "--agent", "bridge", "create-" + tc.sub}, tc.requiredArgs...)
			args = append(args, "--correlation-id=cid-"+tc.sub, "--idempotency-key=idem-"+tc.sub, "--request-id=rid-"+tc.sub)
			_, stderr, err, _ := capturedRun(t, args)
			if err != nil {
				t.Fatalf("dry run returned error: %v", err)
			}
			for _, want := range []string{
				"  Correlation-Id: cid-" + tc.sub + "\n",
				"  Idempotency-Key: idem-" + tc.sub + "\n",
				"  Request-Id: rid-" + tc.sub + "\n",
			} {
				if !strings.Contains(stderr, want) {
					t.Errorf("dry run missing %q\nstderr:\n%s", want, stderr)
				}
			}
		})
	}
}

func TestBridgeCreateHeaderFlagParityWithGenerated(t *testing.T) {
	root := RootCmd()
	for _, path := range [][]string{
		{"bridge", "create-tan"},
		{"bridge", "create-speedchex"},
		{"bridge", "create-bank-account-paykey"},
		{"charges", "create"},
	} {
		cmd, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		for _, name := range []string{"correlation-id", "idempotency-key", "request-id"} {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("%s is missing --%s header flag", strings.Join(path, " "), name)
			}
		}
	}
}
