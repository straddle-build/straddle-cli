//go:build !unix

package cli

import "testing"

func makeFIFO(*testing.T, string) string { return "" }
