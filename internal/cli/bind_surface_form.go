// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/client"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

// uploadContentTypes maps file extensions to the content types a contract's
// multipart encoding may accept. A file flag's Enum narrows these to the
// types its operation allows.
var uploadContentTypes = map[string]string{
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".pdf":  "application/pdf",
	".png":  "image/png",
}

// formFile validates a --file path before any request is built and returns
// the part to stream. The file is only stat'ed and opened, never read, so a
// FIFO or device fails here without blocking.
func formFile(cmd *cobra.Command, definition surface.Flag, path string) (client.FormFile, error) {
	fail := func(format string, args ...any) (client.FormFile, error) {
		return client.FormFile{}, usageErr(fmt.Errorf("--%s %s: "+format, append([]any{definition.Name, path}, args...)...))
	}
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fail("no such file")
	case err != nil:
		return fail("cannot be read: %v", errors.Unwrap(err))
	case info.IsDir():
		return fail("is a directory, not a regular file")
	case !info.Mode().IsRegular():
		return fail("is not a regular file")
	case info.Size() == 0:
		return fail("file is empty")
	}
	if limit := maxUploadBytes(cmd, definition); limit > 0 && info.Size() > limit {
		return fail("file is %d bytes; the maximum is %d bytes", info.Size(), limit)
	}
	contentType, err := uploadContentType(definition, path)
	if err != nil {
		return fail("%v", err)
	}
	file, err := os.Open(path) // #nosec G304 -- the user names the file to upload.
	if err != nil {
		return fail("cannot be read: %v", errors.Unwrap(err))
	}
	_ = file.Close()
	return client.FormFile{
		Field:       definition.Key,
		Path:        path,
		FileName:    filepath.Base(path),
		ContentType: contentType,
	}, nil
}

func uploadContentType(definition surface.Flag, path string) (string, error) {
	contentType := uploadContentTypes[strings.ToLower(filepath.Ext(path))]
	var supported []string
	for extension, candidate := range uploadContentTypes {
		for _, allowed := range definition.Enum {
			if candidate == allowed {
				supported = append(supported, extension)
				if candidate == contentType {
					return contentType, nil
				}
			}
		}
	}
	sort.Strings(supported)
	return "", fmt.Errorf("unsupported file type %q; supported: %s", filepath.Ext(path), strings.Join(supported, ", "))
}

func maxUploadBytes(cmd *cobra.Command, definition surface.Flag) int64 {
	flag := cmd.Flags().Lookup(definition.Name)
	if flag == nil || len(flag.Annotations["straddle:max-bytes"]) == 0 {
		return 0
	}
	limit, _ := strconv.ParseInt(flag.Annotations["straddle:max-bytes"][0], 10, 64)
	return limit
}

// hasFormFlag reports a multipart/form-data surface, which streams its file
// flags instead of taking a JSON body or --stdin.
func hasFormFlag(s surface.Surface) bool {
	for _, flag := range s.Flags {
		if flag.In == surface.InForm {
			return true
		}
	}
	return false
}
