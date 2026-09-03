// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("representatives.get-unmasked-representative", newRepresentativesGetUnmaskedRepresentativeCmd)
	registerSurface(surface.Surface{
		Endpoint:    "representatives.get-unmasked-representative",
		OperationID: "getUnmaskedRepresentative",
		Method:      "GET",
		Path:        "/v1/representatives/{representative_id}/unmask",
		PathParams:  []string{"representative_id"},
		Flags: []surface.Flag{
			{
				Name:        "correlation-id",
				In:          surface.InHeader,
				Key:         "Correlation-Id",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated identifier for tracing a series of related requests.",
			},
			{
				Name:        "request-id",
				In:          surface.InHeader,
				Key:         "Request-Id",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated identifier for tracing one request.",
			},
		},
		HasBody:              false,
		BodyRequired:         false,
		AcceptsAccountHeader: false,
		ReadOnly:             true,
	})
}

func newRepresentativesGetUnmaskedRepresentativeCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "representatives.get-unmasked-representative",
		OperationID: "getUnmaskedRepresentative",
		Method:      "GET",
		Path:        "/v1/representatives/{representative_id}/unmask",
		PathParams:  []string{"representative_id"},
		Flags: []surface.Flag{
			{
				Name:        "correlation-id",
				In:          surface.InHeader,
				Key:         "Correlation-Id",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated identifier for tracing a series of related requests.",
			},
			{
				Name:        "request-id",
				In:          surface.InHeader,
				Key:         "Request-Id",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated identifier for tracing one request.",
			},
		},
		HasBody:              false,
		BodyRequired:         false,
		AcceptsAccountHeader: false,
		ReadOnly:             true,
	}
	cmd := &cobra.Command{
		Use:     "get-unmasked-representative <representative_id>",
		Short:   "Get an unmasked representative",
		Example: "  straddle representatives get-unmasked-representative <representative_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "representatives.get-unmasked-representative",
			"straddle:operation-id": "getUnmaskedRepresentative",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/representatives/{representative_id}/unmask",
			"mcp:read-only":         "true",
		},
	}
	bind := bindSurface(cmd, flags, s)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		req, err := bind(args)
		if err != nil {
			return err
		}
		return executeSurface(cmd, flags, s, req)
	}
	applyOverlay("representatives.get-unmasked-representative", cmd)
	return cmd
}
