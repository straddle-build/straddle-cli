// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/store"
)

func TestWriteThroughCacheUnwrapsObjectEnvelope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeThroughCache(context.Background(), "charges", json.RawMessage(`{"meta":{},"response_type":"object","data":{"id":"ch_123","status":"pending"}}`))
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got, err := db.Get("charges", "ch_123")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"id":"ch_123","status":"pending"}` {
		t.Fatalf("cached object = %s", got)
	}
	if _, err := os.Stat(filepath.Dir(filepath.Join(home, ".local", "share", "straddle", "data.db"))); err != nil {
		t.Fatal(err)
	}
}

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

			db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"))
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
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"))
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
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"))
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
	db, err := store.OpenWithContext(context.Background(), filepath.Join(home, ".local", "share", "straddle", "data.db"))
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
