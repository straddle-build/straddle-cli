// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"bytes"
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

// open returns the streamed body with its exact length and Content-Type.
// Closing the body closes every opened file; net/http closes request bodies
// even when the request fails.
func (m MultipartForm) open() (io.ReadCloser, int64, string, error) {
	var framing bytes.Buffer
	writer := multipart.NewWriter(&framing)
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
