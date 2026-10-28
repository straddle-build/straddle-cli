// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
)

// FormFile is one file part of a multipart/form-data request body.
type FormFile struct {
	Field       string
	Path        string
	FileName    string
	ContentType string
}

// MultipartForm is a request body sent as multipart/form-data. Each attempt
// reopens and streams its files, so a body is never held in memory and a
// retry resends the same bytes.
type MultipartForm struct {
	Files []FormFile
}

// openFormFile is os.Open; tests replace it to observe that every opened
// file is closed.
var openFormFile = os.Open

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"", "\r", "", "\n", "")

// newBoundary returns a fresh, cryptographically random multipart boundary.
// A single request reuses one boundary across every (re)open of its form so
// the framing — and therefore the Content-Type header net/http copies when it
// follows a 307/308 redirect — matches the replayed body byte-for-byte.
// Generating a new boundary per open() instead would hand the redirect hop a
// different boundary than the copied Content-Type, so the server could not
// parse the replayed body.
func newBoundary() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) //nolint:gosec // boundary strength, not a secret
	return "straddle-" + hex.EncodeToString(b[:])
}

// open returns the streamed body with its exact length and Content-Type.
// Closing the body closes every opened file; net/http closes request bodies
// even when the request fails. The boundary is random per call; callers
// that need a stable boundary across replays (redirects, retries) use
// openWithBoundary so the Content-Type and the replayed framing agree.
func (m MultipartForm) open() (io.ReadCloser, int64, string, error) {
	return m.openWithBoundary(newBoundary())
}

// openWithBoundary is open() with a caller-chosen boundary, so every
// (re)open of the same request — the initial send, a 307/308 redirect
// replay via Request.GetBody, or a retry — produces byte-identical framing
// and a matching Content-Type.
func (m MultipartForm) openWithBoundary(boundary string) (io.ReadCloser, int64, string, error) {
	var framing bytes.Buffer
	writer := multipart.NewWriter(&framing)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, 0, "", err
	}
	var readers []io.Reader
	var files []*os.File
	fail := func(err error) (io.ReadCloser, int64, string, error) {
		closeFiles(files)
		return nil, 0, "", err
	}
	var length int64
	for _, part := range m.Files {
		file, err := openFormFile(part.Path)
		if err != nil {
			return fail(err)
		}
		files = append(files, file)
		info, err := file.Stat()
		if err != nil {
			return fail(err)
		}
		if !info.Mode().IsRegular() {
			return fail(fmt.Errorf("%s is not a regular file", part.Path))
		}
		start := framing.Len()
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, quoteEscaper.Replace(part.Field), quoteEscaper.Replace(part.FileName)))
		header.Set("Content-Type", part.ContentType)
		if _, err := writer.CreatePart(header); err != nil {
			return fail(err)
		}
		prefix := append([]byte(nil), framing.Bytes()[start:]...)
		readers = append(readers, bytes.NewReader(prefix), file)
		length += int64(len(prefix)) + info.Size()
	}
	start := framing.Len()
	if err := writer.Close(); err != nil {
		return fail(err)
	}
	trailer := append([]byte(nil), framing.Bytes()[start:]...)
	readers = append(readers, bytes.NewReader(trailer))
	return &formBody{Reader: io.MultiReader(readers...), files: files}, length + int64(len(trailer)), writer.FormDataContentType(), nil
}

type formBody struct {
	io.Reader
	files []*os.File
}

func (b *formBody) Close() error {
	closeFiles(b.files)
	return nil
}

func closeFiles(files []*os.File) {
	for _, file := range files {
		_ = file.Close()
	}
}

// describe prints a dry-run summary of the form without reading any file.
func (m MultipartForm) describe(w io.Writer) {
	fmt.Fprintf(w, "  Body: multipart/form-data\n")
	for _, part := range m.Files {
		size := "unknown size"
		if info, err := os.Stat(part.Path); err == nil {
			size = fmt.Sprintf("%d bytes", info.Size())
		}
		fmt.Fprintf(w, "    %s: %s (%s, %s)\n", part.Field, part.FileName, part.ContentType, size)
	}
}
