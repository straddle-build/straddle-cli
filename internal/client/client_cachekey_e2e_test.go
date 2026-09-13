// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/straddle-build/straddle-cli/internal/config"
)

func TestCacheKey_StaleResponseAcrossTemplateVars(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"path":%q}`, r.URL.Path)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	baseURL := server.URL + "/{shop}"
	alice := New(&config.Config{
		BaseURL:      baseURL,
		TemplateVars: map[string]string{"environment": "sandbox", "shop": "alice"},
	}, time.Second, 0)
	alice.cacheDir = cacheDir
	bob := New(&config.Config{
		BaseURL:      baseURL,
		TemplateVars: map[string]string{"environment": "sandbox", "shop": "bob"},
	}, time.Second, 0)
	bob.cacheDir = cacheDir

	first, err := alice.Get("/list", nil)
	if err != nil || string(first) != `{"path":"/alice/list"}` {
		t.Fatalf("alice Get = %s, %v; want alice's response", first, err)
	}
	second, err := bob.Get("/list", nil)
	if err != nil || string(second) != `{"path":"/bob/list"}` {
		t.Fatalf("bob Get = %s, %v; want bob's response", second, err)
	}
	cached, err := alice.Get("/list", nil)
	if err != nil || string(cached) != `{"path":"/alice/list"}` {
		t.Fatalf("cached alice Get = %s, %v; want alice's response", cached, err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("server requests = %d, want 2 with the repeated request cached", got)
	}
}

func TestCacheKey_UnresolvedDoesNotReceiveResolvedCachedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"shop":"alice"}`)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	baseURL := server.URL + "/{shop}"
	resolved := New(&config.Config{
		BaseURL:      baseURL,
		TemplateVars: map[string]string{"environment": "sandbox", "shop": "alice"},
	}, time.Second, 0)
	resolved.cacheDir = cacheDir
	body, err := resolved.Get("/list", nil)
	if err != nil || string(body) != `{"shop":"alice"}` {
		t.Fatalf("resolved Get = %s, %v; want alice's response", body, err)
	}

	unresolved := New(&config.Config{
		BaseURL:      baseURL,
		TemplateVars: map[string]string{"environment": "sandbox"},
	}, time.Second, 0)
	unresolved.cacheDir = cacheDir
	_, err = unresolved.Get("/list", nil)
	var templateErr *TemplateVarError
	if !errors.As(err, &templateErr) {
		t.Fatalf("unresolved Get error = %v, want TemplateVarError", err)
	}
	if !slices.Equal(templateErr.Names, []string{"shop"}) {
		t.Fatalf("unresolved names = %v, want [shop]", templateErr.Names)
	}
}
