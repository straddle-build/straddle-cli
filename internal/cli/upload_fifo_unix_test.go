//go:build unix

package cli

import (
	"path/filepath"
	"syscall"
	"testing"
)

func makeFIFO(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "pipe.pdf")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	return path
}
