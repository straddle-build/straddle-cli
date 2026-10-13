// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/straddle-build/straddle-cli/internal/config"
)

// redirectProofBytes is binary content that includes multipart-looking
// bytes, so a framing mistake shows up as a byte mismatch on replay.
var redirectProofBytes = []byte("%PDF-1.4\n\x00\xff\r\n--not-a-boundary\r\nredirect-marker-7f3a\n%%EOF\n")

func newRedirectMultipartClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c := New(&config.Config{BaseURL: baseURL}, 5*time.Second, 0)
	c.NoCache = true
	return c
}

func newUploadForm(t *testing.T) (MultipartForm, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proof.pdf")
	if err := os.WriteFile(path, redirectProofBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	return MultipartForm{Files: []FormFile{{
		Field:       "File",
		Path:        path,
		FileName:    "proof.pdf",
		ContentType: "application/pdf",
	}}}, path
}

// allClosed asserts every recorded file handle has been closed.
func allClosed(t *testing.T, files []*os.File) {
	t.Helper()
	if len(files) == 0 {
		t.Fatal("expected at least one opened file handle")
	}
	for i, file := range files {
		if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("file %d (%s) still open: Stat error = %v", i, file.Name(), err)
		}
	}
}

type capturedPart struct {
	method        string
	contentType   string
	contentLength int64
	field         string
	fileName      string
	partType      string
	body          []byte
}

// redirectUploadServer answers a 3xx at uploadPath pointing at targetPath.
// The target records the single multipart part it receives.
func redirectUploadServer(t *testing.T, code int, uploadPath, targetPath string) (*httptest.Server, func() (capturedPart, int)) {
	t.Helper()
	var mu sync.Mutex
	var got capturedPart
	var targetHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == uploadPath {
			_, _ = io.Copy(io.Discard, r.Body)
			http.Redirect(w, r, targetPath, code)
			return
		}
		atomic.AddInt32(&targetHits, 1)
		c := capturedPart{
			method:        r.Method,
			contentType:   r.Header.Get("Content-Type"),
			contentLength: r.ContentLength,
		}
		if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType == "multipart/form-data" {
			if reader, err := r.MultipartReader(); err == nil {
				for {
					part, err := reader.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Errorf("target NextPart: %v", err)
						break
					}
					c.field, c.fileName, c.partType = part.FormName(), part.FileName(), part.Header.Get("Content-Type")
					c.body, _ = io.ReadAll(part)
				}
			}
		} else {
			c.body, _ = io.ReadAll(r.Body)
		}
		mu.Lock()
		got = c
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)
	return server, func() (capturedPart, int) {
		return got, int(atomic.LoadInt32(&targetHits))
	}
}

// redirectNoLocationServer replies with a 3xx that carries NO Location
// header, which net/http returns to the caller unchanged with a nil error.
func redirectNoLocationServer(t *testing.T, code int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestMultipartUploadRedirectReplaysBody proves a 307/308 redirect replays
// the streamed multipart body byte-for-byte: net/http needs GetBody to
// resend the body, and the request keeps its POST method so the upload
// reaches the redirect target. Before the fix, GetBody was nil and the
// 3xx was handed back untouched (then misclassified as a success).
func TestMultipartUploadRedirectReplaysBody(t *testing.T) {
	for _, code := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			opened := trackOpenedFormFiles(t)
			server, capture := redirectUploadServer(t, code, "/v1/charges/x/authorization", "/v1/charges/x/authorization/finalize")
			c := newRedirectMultipartClient(t, server.URL)
			form, _ := newUploadForm(t)

			data, status, err := c.DoWithValues(http.MethodPost, "/v1/charges/x/authorization", nil, form, nil)
			if err != nil {
				t.Fatalf("upload after %d: %v", code, err)
			}
			if status != http.StatusCreated {
				t.Fatalf("status = %d, want %d", status, http.StatusCreated)
			}
			if !bytes.Equal(data, []byte(`{"ok":true}`)) {
				t.Fatalf("response body = %s, want {\"ok\":true}", data)
			}

			got, hits := capture()
			if hits != 1 {
				t.Fatalf("target received %d requests, want 1", hits)
			}
			if got.method != http.MethodPost {
				t.Fatalf("target saw method %q, want POST (307/308 preserve the method)", got.method)
			}
			if got.field != "File" || got.fileName != "proof.pdf" || got.partType != "application/pdf" {
				t.Fatalf("target part = field %q, filename %q, type %q", got.field, got.fileName, got.partType)
			}
			if !bytes.Equal(got.body, redirectProofBytes) {
				t.Fatalf("replayed part bytes = %q, want the whole proof", got.body)
			}
			// The original attempt opens the file, then GetBody reopens it for
			// the redirect hop. Both must be closed by net/http.
			requireAllClosed(t, opened(), 2)
		})
	}
}

// TestMultipartUploadRedirectToBodylessGETIsRefused proves a 301/302/303
// redirect — which net/http rewrites to a bodyless GET, dropping the file
// — is refused instead of followed. Without the guard the target would
// answer 200 to that GET and the client would report a success while the
// file was never delivered.
func TestMultipartUploadRedirectToBodylessGETIsRefused(t *testing.T) {
	for _, code := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			opened := trackOpenedFormFiles(t)
			server, capture := redirectUploadServer(t, code, "/v1/charges/x/authorization", "/v1/charges/x/authorization/finalize")
			c := newRedirectMultipartClient(t, server.URL)
			form, _ := newUploadForm(t)

			data, status, err := c.DoWithValues(http.MethodPost, "/v1/charges/x/authorization", nil, form, nil)
			if err == nil {
				t.Fatalf("upload after %d returned (body %s, status %d, nil error); want a refused-redirect error", code, data, status)
			}
			if !containsSubstring(err.Error(), "refused to follow redirect of multipart upload") {
				t.Fatalf("error = %v, want a multipart redirect refusal", err)
			}
			// The bodyless GET must never be issued to the target.
			if _, hits := capture(); hits != 0 {
				t.Fatalf("target received a request; the bodyless GET should never have been sent")
			}
			allClosed(t, opened())
		})
	}
}

// TestMultipartUploadReturned3xxNotTreatedAsSuccess proves a 3xx that
// reaches the client (here a 307 with no Location header, which net/http
// returns unchanged with a nil error) is reported as an APIError instead
// of falling through the success branch. Before the fix the < 400
// branch classified it as a successful upload with an empty/HTML body.
func TestMultipartUploadReturned3xxNotTreatedAsSuccess(t *testing.T) {
	opened := trackOpenedFormFiles(t)
	server := redirectNoLocationServer(t, http.StatusTemporaryRedirect, "<html>moved</html>")
	c := newRedirectMultipartClient(t, server.URL)
	form, _ := newUploadForm(t)

	data, status, err := c.DoWithValues(http.MethodPost, "/v1/charges/x/authorization", nil, form, nil)
	if err == nil {
		t.Fatalf("a returned 307 was classified as success (body %q, status %d); want an error", data, status)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError for a returned 3xx", err)
	}
	if apiErr.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("APIError status = %d, want %d", apiErr.StatusCode, http.StatusTemporaryRedirect)
	}
	allClosed(t, opened())
}

// TestJSONPostRedirectReplaysBody pins the pre-existing JSON-path behavior
// (a 307 replays the marshaled body) so the multipart fix cannot regress it.
func TestJSONPostRedirectReplaysBody(t *testing.T) {
	var mu sync.Mutex
	var targetMethod string
	var targetBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/authorize" {
			_, _ = io.Copy(io.Discard, r.Body)
			http.Redirect(w, r, "/finalize", http.StatusTemporaryRedirect)
			return
		}
		mu.Lock()
		targetMethod = r.Method
		targetBody, _ = io.ReadAll(r.Body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	c := newRedirectMultipartClient(t, server.URL)
	data, status, err := c.DoWithValues(http.MethodPost, "/authorize", nil, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatalf("JSON POST after 307: %v", err)
	}
	if status != http.StatusCreated || !bytes.Equal(data, []byte(`{"ok":true}`)) {
		t.Fatalf("JSON POST result = (%s, %d), want ({\"ok\":true}, 201)", data, status)
	}
	mu.Lock()
	defer mu.Unlock()
	if targetMethod != http.MethodPost {
		t.Fatalf("target method = %q, want POST (JSON 307 preserves the method and replays the body)", targetMethod)
	}
	if !bytes.Equal(targetBody, []byte(`{"x":1}`)) {
		t.Fatalf("target body = %q, want {\"x\":1}", targetBody)
	}
}

// TestMultipartUploadDirectSuccessUnchanged guards against regressing the
// non-redirected happy path: a 2xx answer is still a success with the file
// delivered and exactly one (re)open that is closed.
func TestMultipartUploadDirectSuccessUnchanged(t *testing.T) {
	opened := trackOpenedFormFiles(t)
	var got capturedPart
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = capturedPart{
			method:        r.Method,
			contentType:   r.Header.Get("Content-Type"),
			contentLength: r.ContentLength,
		}
		if reader, err := r.MultipartReader(); err == nil {
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("NextPart: %v", err)
					break
				}
				got.field, got.fileName, got.partType = part.FormName(), part.FileName(), part.Header.Get("Content-Type")
				got.body, _ = io.ReadAll(part)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	c := newRedirectMultipartClient(t, server.URL)
	form, _ := newUploadForm(t)

	data, status, err := c.DoWithValues(http.MethodPost, "/v1/charges/x/authorization", nil, form, nil)
	if err != nil || status != http.StatusCreated {
		t.Fatalf("direct upload = (%d, %v), want (201, nil)", status, err)
	}
	if !bytes.Equal(data, []byte(`{"ok":true}`)) {
		t.Fatalf("response body = %s, want {\"ok\":true}", data)
	}
	if !bytes.Equal(got.body, redirectProofBytes) {
		t.Fatalf("part bytes = %q, want the whole proof", got.body)
	}
	requireAllClosed(t, opened(), 1)
}

// containsSubstring is a tiny helper to avoid importing strings in this
// test's hot path.
func containsSubstring(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
