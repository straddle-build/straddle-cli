// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeNeutralFixture writes a regular file under a conventional name so the
// test never depends on the filesystem allowing control characters in names.
// The malicious value is injected only into FormFile.FileName/Field, which is
// what open() writes into the part's MIME headers.
func writeNeutralFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proof.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4 crlf test"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func firstPart(t *testing.T, raw []byte, contentType string) *multipart.Part {
	t.Helper()
	boundary := strings.TrimPrefix(contentType, "multipart/form-data; boundary=")
	r := multipart.NewReader(strings.NewReader(string(raw)), boundary)
	p, err := r.NextPart()
	if err != nil {
		t.Fatalf("NextPart: %v\nraw:\n%s", err, raw)
	}
	return p
}

// TestMultipartStripsCRLFInPartHeaders verifies that a CR/LF in the part's
// FileName or Field cannot inject additional MIME part headers onto the wire.
// The primary guard is formFile rejecting such names before the request is
// built; open() strips CR/LF from header values as defense-in-depth so a future
// caller or surface change cannot reintroduce the injection.
func TestMultipartStripsCRLFInPartHeaders(t *testing.T) {
	path := writeNeutralFixture(t)
	cases := []struct {
		name     string
		field    string
		fileName string
	}{
		{"LF_in_filename", "File", "report\nX-Injected: evil.pdf"},
		{"CR_in_filename", "File", "report\rX-Injected: evil.pdf"},
		{"CRLF_in_filename", "File", "report\r\nX-Injected: evil.pdf"},
		{"LF_content_type_override", "File", "report\nContent-Type: text/html.pdf"},
		{"LF_in_field", "File\nX-Injected: evil", "report.pdf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			form := MultipartForm{Files: []FormFile{
				{Field: tc.field, Path: path, FileName: tc.fileName, ContentType: "application/pdf"},
			}}
			body, _, contentType, err := form.open()
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer body.Close()
			raw, _ := io.ReadAll(body)

			p := firstPart(t, raw, contentType)
			defer p.Close()

			// No injected header line should be visible to a server parser.
			if got := p.Header.Get("X-Injected"); got != "" {
				t.Errorf("server parser saw injected X-Injected=%q\nraw:\n%s", got, raw)
			}
			// The intended Content-Type must win; under textproto first-value-wins
			// an injected "Content-Type:" line preceding the real one would
			// override it.
			if got, want := p.Header.Get("Content-Type"), "application/pdf"; got != want {
				t.Errorf("Content-Type = %q, want %q (injection overrode it)\nraw:\n%s", got, want, raw)
			}
			// The part must have exactly the two intended header fields.
			if got, want := len(p.Header), 2; got != want {
				t.Errorf("part has %d header fields, want %d\nraw:\n%s", got, want, raw)
			}
			// The on-wire Content-Disposition value must contain no CR/LF, so it
			// cannot be split into additional header lines.
			if cd := p.Header.Get("Content-Disposition"); strings.ContainsAny(cd, "\r\n") {
				t.Errorf("Content-Disposition contains CR/LF: %q\nraw:\n%s", cd, raw)
			}
		})
	}
}

// TestMultipartPreservesOrdinaryFilenames is a regression guard that the CR/LF
// scrubbing does not mangle ordinary filenames or the existing backslash/quote
// escaping.
func TestMultipartPreservesOrdinaryFilenames(t *testing.T) {
	path := writeNeutralFixture(t)
	cases := []string{
		"report.pdf",
		"signed proof of upload.pdf",
		`weird\name.pdf`,
		`q"uote".pdf`,
		`a\"b.pdf`,
	}
	for _, fileName := range cases {
		t.Run(fileName, func(t *testing.T) {
			form := MultipartForm{Files: []FormFile{
				{Field: "File", Path: path, FileName: fileName, ContentType: "application/pdf"},
			}}
			body, _, contentType, err := form.open()
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer body.Close()
			raw, _ := io.ReadAll(body)

			p := firstPart(t, raw, contentType)
			defer p.Close()
			if got := p.FileName(); got != fileName {
				t.Errorf("FileName = %q, want %q (escaping mangled it)\nraw:\n%s", got, fileName, raw)
			}
			if got, want := p.Header.Get("Content-Type"), "application/pdf"; got != want {
				t.Errorf("Content-Type = %q, want %q\nraw:\n%s", got, want, raw)
			}
			if got, want := len(p.Header), 2; got != want {
				t.Errorf("part has %d header fields, want %d\nraw:\n%s", got, want, raw)
			}
		})
	}
}
