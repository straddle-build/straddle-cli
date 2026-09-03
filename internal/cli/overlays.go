// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import "github.com/spf13/cobra"

var endpointOverlays = map[string]func(*cobra.Command){}

func applyOverlay(endpoint string, cmd *cobra.Command) {
	if f, ok := endpointOverlays[endpoint]; ok {
		f(cmd)
	}
}
