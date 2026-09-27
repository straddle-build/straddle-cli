// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/straddle-build/straddle-cli/internal/config"
)

// trackOpenedFormFiles records every file the multipart body opens. Not
// parallel-safe: it swaps a package variable.
func trackOpenedFormFiles(t *testing.T) func() []*os.File {
	t.Helper()
	var mu sync.Mutex
	var opened []*os.File
	openFormFile = func(path string) (*os.File, error) {
		file, err := os.Open(path) // #nosec G304 -- test fixture paths.
		if err == nil {
			mu.Lock()
			opened = append(opened, file)
			mu.Unlock()
		}
		return file, err
	}
	t.Cleanup(func() { openFormFile = os.Open })
	return func() []*os.File {
		mu.Lock()
		defer mu.Unlock()
		return append([]*os.File(nil), opened...)
	}
}

func requireAllClosed(t *testing.T, files []*os.File, want int) {
	t.Helper()
	if len(files) != want {
		t.Fatalf("opened %d files, want %d", len(files), want)
	}
	for i, file := range files {
		if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("file %d (%s) still open: Stat error = %v", i, file.Name(), err)
		}
	}
}

func writeFormFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proof.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4 close test"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMultipartUploadClosesFileAfterCompletedRequest(t *testing.T) {
	opened := trackOpenedFormFiles(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	c := New(&config.Config{BaseURL: server.URL}, 5*time.Second, 0)
	c.NoCache = true

	form := MultipartForm{Files: []FormFile{{Field: "File", Path: writeFormFixture(t), FileName: "proof.pdf", ContentType: "application/pdf"}}}
	if _, status, err := c.DoWithValues(http.MethodPost, "/v1/charges/x/authorization", nil, form, nil); err != nil || status != http.StatusCreated {
		t.Fatalf("upload = %d, %v", status, err)
	}
	requireAllClosed(t, opened(), 1)
}

func TestMultipartUploadClosesFileWhenEveryAttemptFails(t *testing.T) {
	opened := trackOpenedFormFiles(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	unreachable := "http://" + listener.Addr().String()
	_ = listener.Close()
	c := New(&config.Config{BaseURL: unreachable}, time.Second, 0)
	c.NoCache = true

	form := MultipartForm{Files: []FormFile{{Field: "File", Path: writeFormFixture(t), FileName: "proof.pdf", ContentType: "application/pdf"}}}
	if _, _, err := c.DoWithValues(http.MethodPost, "/v1/charges/x/authorization", nil, form, nil); err == nil {
		t.Fatal("upload to a closed port succeeded")
	}
	// One open per attempt: the first try plus three retries.
	requireAllClosed(t, opened(), 4)
}

func TestMultipartOpenClosesEarlierFilesWhenALaterPartFails(t *testing.T) {
	good := writeFormFixture(t)
	for name, bad := range map[string]string{
		"missing":   filepath.Join(t.TempDir(), "absent.pdf"),
		"directory": t.TempDir(),
	} {
		t.Run(name, func(t *testing.T) {
			opened := trackOpenedFormFiles(t)
			form := MultipartForm{Files: []FormFile{
				{Field: "A", Path: good, FileName: "a.pdf", ContentType: "application/pdf"},
				{Field: "B", Path: bad, FileName: "b.pdf", ContentType: "application/pdf"},
			}}
			if body, _, _, err := form.open(); err == nil {
				_ = body.Close()
				t.Fatal("open succeeded with an unusable part")
			}
			files := opened()
			if name == "missing" {
				requireAllClosed(t, files, 1)
			} else {
				requireAllClosed(t, files, 2)
			}
		})
	}
}
