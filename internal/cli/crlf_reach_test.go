// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

// formFileCRLFDefinition is a contract-shaped file flag that accepts only PDFs,
// matching the real upload-authorization-proof --file surface.
func formFileCRLFDefinition() surface.Flag {
	return surface.Flag{Name: "file", Key: "File", In: surface.InForm, Kind: surface.KindFile, Enum: []string{"application/pdf"}}
}

func crlfFormFileCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("file", "", "")
	_ = cmd.Flags().SetAnnotation("file", "straddle:max-bytes", []string{"0"})
	return cmd
}

// TestFormFileRejectsCRLFInFilename confirms the primary guard: a filename
// containing CR/LF that would otherwise bypass the extension allow-list (the
// newline lands before a trailing allowed extension) is rejected with a clear
// usage error before any request is built.
func TestFormFileRejectsCRLFInFilename(t *testing.T) {
	def := formFileCRLFDefinition()
	cases := []string{
		"report\nX-Injected: evil.pdf",        // LF before allowed ext (the documented bypass)
		"report\rX-Injected: evil.pdf",        // CR before allowed ext
		"report\r\nX-Injected: evil.pdf",      // CRLF before allowed ext
		"report\nContent-Type: textplain.pdf", // Content-Type override variant (no slash: "/" is not allowed in a basename)
	}
	for _, fname := range cases {
		label := strings.ReplaceAll(strings.ReplaceAll(fname, "\n", `\n`), "\r", `\r`)
		t.Run(label, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), fname)
			if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
				t.Skipf("filesystem does not allow control characters in names: %v", err)
			}
			_, err := formFile(crlfFormFileCmd(), def, path)
			if err == nil {
				t.Fatalf("formFile accepted filename with control character: %q", fname)
			}
			if !strings.Contains(err.Error(), "filename may not contain newline or carriage-return characters") {
				t.Fatalf("error = %v, want the CRLF rejection message", err)
			}
			if ExitCode(err) != 2 {
				t.Fatalf("exit code = %d, want 2 (usage error)", ExitCode(err))
			}
		})
	}
}

// TestFormFileAcceptsOrdinaryFilenames is a regression guard that the CRLF
// check does not reject conventional filenames (including spaces and dots).
func TestFormFileAcceptsOrdinaryFilenames(t *testing.T) {
	def := formFileCRLFDefinition()
	ordinary := []string{"report.pdf", "signed proof of upload.pdf", "Q1 2026.pdf", "a.b.c.pdf"}
	for _, fname := range ordinary {
		t.Run(fname, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), fname)
			if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
				t.Skipf("fs: %v", err)
			}
			ff, err := formFile(crlfFormFileCmd(), def, path)
			if err != nil {
				t.Fatalf("formFile rejected ordinary filename %q: %v", fname, err)
			}
			if ff.FileName != fname {
				t.Fatalf("FileName = %q, want %q", ff.FileName, fname)
			}
			if ff.ContentType != "application/pdf" {
				t.Fatalf("ContentType = %q, want application/pdf", ff.ContentType)
			}
		})
	}
}

// TestUploadAuthorizationProofRejectsCRLFFilenameEndToEnd exercises the full
// CLI path: a --file argument whose base name contains a newline is rejected
// with exit code 2 and no HTTP request is sent.
func TestUploadAuthorizationProofRejectsCRLFFilenameEndToEnd(t *testing.T) {
	server := newUploadServer(t)
	fname := "report\nContent-Type: textplain.pdf"
	path := filepath.Join(t.TempDir(), fname)
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Skipf("filesystem does not allow control characters in names: %v", err)
	}
	_, _, err := runRootForAPITest(t, []string{"--agent", "charges", "upload-authorization-proof", uploadTestID, "--file", path}, "")
	if err == nil {
		t.Fatal("upload with a CRLF filename succeeded")
	}
	if !strings.Contains(err.Error(), "filename may not contain newline or carriage-return characters") {
		t.Fatalf("error = %v, want the CRLF rejection message", err)
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2 (usage error)", ExitCode(err))
	}
	if n := server.requests.Load(); n != 0 {
		t.Fatalf("sent %d request(s) for a CRLF filename; rejection must happen before any request", n)
	}
}
