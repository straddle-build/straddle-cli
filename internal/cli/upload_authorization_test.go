package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/surface"
)

const uploadTestID = "550e8400-e29b-41d4-a716-446655440000"

// uploadProof is binary content that includes multipart-looking bytes, so a
// framing mistake shows up as a byte mismatch.
var uploadProof = []byte("%PDF-1.4\n\x00\xff\r\n--not-a-boundary\r\nupload-marker-7f3a\n%%EOF\n")

type receivedUpload struct {
	path          string
	contentLength int64
	field         string
	fileName      string
	contentType   string
	body          []byte
}

type uploadServer struct {
	url      string
	requests atomic.Int32
	mu       sync.Mutex
	received []receivedUpload
}

// newUploadServer answers each upload with the given statuses in order,
// then 201, recording every multipart part it receives.
func newUploadServer(t *testing.T, statuses ...int) *uploadServer {
	t.Helper()
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_API_KEY", "test_key")
	s := &uploadServer{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(s.requests.Add(1))
		got := receivedUpload{path: r.URL.Path, contentLength: r.ContentLength}
		if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType != "multipart/form-data" {
			t.Errorf("Content-Type = %q, want multipart/form-data", r.Header.Get("Content-Type"))
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Errorf("MultipartReader: %v", err)
		} else {
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("NextPart: %v", err)
					break
				}
				got.field, got.fileName, got.contentType = part.FormName(), part.FileName(), part.Header.Get("Content-Type")
				got.body, _ = io.ReadAll(part)
			}
		}
		s.mu.Lock()
		s.received = append(s.received, got)
		s.mu.Unlock()
		status := http.StatusCreated
		if n <= len(statuses) {
			status = statuses[n-1]
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status >= http.StatusBadRequest {
			_, _ = w.Write([]byte(`{"error":{"title":"File signature does not match its extension"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"meta":{"api_request_id":"r1"},"response_type":"object","data":{"id":"` + uploadTestID + `","status":"created"}}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("STRADDLE_BASE_URL", server.URL)
	s.url = server.URL
	return s
}

func writeUpload(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUploadAuthorizationProofStreamsFileToResource(t *testing.T) {
	for _, resource := range []string{"charges", "payouts"} {
		t.Run(resource, func(t *testing.T) {
			server := newUploadServer(t)
			path := writeUpload(t, "signed proof.pdf", uploadProof)

			stdout, stderr, err := runRootForAPITest(t, []string{"--agent", resource, "upload-authorization-proof", uploadTestID, "--file", path}, "")
			if err != nil {
				t.Fatalf("upload: %v\nstderr: %s", err, stderr)
			}
			if len(server.received) != 1 {
				t.Fatalf("requests = %d, want 1", len(server.received))
			}
			got := server.received[0]
			if want := "/v1/" + resource + "/" + uploadTestID + "/authorization"; got.path != want {
				t.Fatalf("path = %q, want %q", got.path, want)
			}
			if got.field != "File" || got.fileName != "signed proof.pdf" || got.contentType != "application/pdf" {
				t.Fatalf("part = field %q, filename %q, type %q", got.field, got.fileName, got.contentType)
			}
			if !bytes.Equal(got.body, uploadProof) {
				t.Fatalf("part bytes = %q, want %q", got.body, uploadProof)
			}
			if got.contentLength <= int64(len(uploadProof)) {
				t.Fatalf("Content-Length = %d, want the exact framed length, not chunked", got.contentLength)
			}

			var envelope map[string]any
			if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
				t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
			}
			if envelope["action"] != "upload-authorization-proof" || envelope["resource"] != resource ||
				envelope["status"] != float64(http.StatusCreated) || envelope["success"] != true {
				t.Fatalf("envelope = %v", envelope)
			}
			if strings.Contains(stdout+stderr, "upload-marker-7f3a") {
				t.Fatalf("output contains file contents:\n%s\n%s", stdout, stderr)
			}
		})
	}
}

func TestUploadAuthorizationProofRetryResendsWholeFile(t *testing.T) {
	server := newUploadServer(t, http.StatusServiceUnavailable)
	path := writeUpload(t, "proof.pdf", uploadProof)

	var err error
	stderr := captureStderr(t, func() {
		_, _, err = runRootForAPITest(t, []string{"--agent", "charges", "upload-authorization-proof", uploadTestID, "--file", path}, "")
	})
	if err != nil {
		t.Fatalf("upload after one 503: %v\nstderr: %s", err, stderr)
	}
	if len(server.received) != 2 {
		t.Fatalf("requests = %d, want the 503 attempt and one retry", len(server.received))
	}
	for i, got := range server.received {
		if !bytes.Equal(got.body, uploadProof) || got.contentType != "application/pdf" {
			t.Fatalf("attempt %d sent %q as %q, want the whole file as application/pdf", i+1, got.body, got.contentType)
		}
	}
}

// The API, not the CLI, checks that the bytes match the extension. A PDF
// named .png passes local checks, and the server's rejection must surface
// as an API error without a retry or a success envelope.
func TestUploadAuthorizationProofSurfacesServerSignatureRejection(t *testing.T) {
	server := newUploadServer(t, http.StatusBadRequest)
	path := writeUpload(t, "proof.png", uploadProof)

	stdout, _, err := runRootForAPITest(t, []string{"--agent", "charges", "upload-authorization-proof", uploadTestID, "--file", path}, "")
	if err == nil {
		t.Fatalf("rejected upload succeeded: %s", stdout)
	}
	if ExitCode(err) != 5 || !strings.Contains(err.Error(), "HTTP 400") || !strings.Contains(err.Error(), "File signature does not match its extension") {
		t.Fatalf("error = %v (exit %d), want the API's HTTP 400 message with exit 5", err, ExitCode(err))
	}
	if len(server.received) != 1 || server.received[0].contentType != "image/png" {
		t.Fatalf("requests = %+v, want one attempt sent as image/png", server.received)
	}
	if strings.Contains(stdout, `"success"`) {
		t.Fatalf("rejected upload printed an envelope: %s", stdout)
	}
}

func TestUploadAuthorizationProofDryRunSendsNothing(t *testing.T) {
	server := newUploadServer(t)
	path := writeUpload(t, "proof.pdf", uploadProof)

	stdout, stderr, err, _ := capturedRun(t, []string{"--dry-run", "--agent", "payouts", "upload-authorization-proof", uploadTestID, "--file", path})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if n := server.requests.Load(); n != 0 {
		t.Fatalf("dry run sent %d requests", n)
	}
	if !strings.Contains(stderr, "File: proof.pdf (application/pdf, ") {
		t.Fatalf("dry run does not describe the file:\n%s", stderr)
	}
	if strings.Contains(stdout+stderr, "upload-marker-7f3a") {
		t.Fatalf("dry run printed file contents:\n%s\n%s", stdout, stderr)
	}
}

func TestUploadAuthorizationProofRejectsFilesBeforeRequest(t *testing.T) {
	dir := t.TempDir()
	unreadable := writeUpload(t, "locked.pdf", uploadProof)
	if err := os.Chmod(unreadable, 0); err != nil {
		t.Fatal(err)
	}
	oversize := filepath.Join(dir, "large.pdf")
	if err := os.WriteFile(oversize, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(oversize, 10<<20+1); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "missing", path: filepath.Join(dir, "absent.pdf"), want: "no such file"},
		{name: "directory", path: dir, want: "is a directory, not a regular file"},
		{name: "empty", path: writeUpload(t, "empty.pdf", nil), want: "file is empty"},
		{name: "over limit", path: oversize, want: "file is 10485761 bytes; the maximum is 10485760 bytes"},
		{name: "unsupported type", path: writeUpload(t, "proof.txt", uploadProof), want: `unsupported file type ".txt"; supported: .doc, .docx, .jpeg, .jpg, .pdf, .png`},
		{name: "unreadable", path: unreadable, want: "cannot be read: permission denied"},
	}
	if fifo := makeFIFO(t, dir); fifo != "" {
		tests = append(tests, struct{ name, path, want string }{name: "fifo", path: fifo, want: "is not a regular file"})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "unreadable" && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("file permissions do not deny this user")
			}
			server := newUploadServer(t)
			_, _, err := runRootForAPITest(t, []string{"--agent", "charges", "upload-authorization-proof", uploadTestID, "--file", tc.path}, "")
			if err == nil || !strings.Contains(err.Error(), "--file "+tc.path+": "+tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if ExitCode(err) != 2 {
				t.Fatalf("exit code = %d, want 2", ExitCode(err))
			}
			if n := server.requests.Load(); n != 0 {
				t.Fatalf("sent %d requests for a rejected file", n)
			}
		})
	}
}

func TestUploadContentTypeRequiresContractEnum(t *testing.T) {
	definition := surface.Flag{
		Name: "file",
		In:   surface.InForm,
		Key:  "File",
		Kind: surface.KindFile,
	}
	for _, file := range []string{"proof.pdf", "proof.octet", "proof"} {
		if _, err := uploadContentType(definition, file); err == nil {
			t.Fatalf("uploadContentType(%q) with empty Enum succeeded, want rejection", file)
		}
	}
}
