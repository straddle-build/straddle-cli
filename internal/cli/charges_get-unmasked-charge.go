// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("charges.get-unmasked-charge", newChargesGetUnmaskedChargeCmd)
	registerSurface(surface.Surface{
		Endpoint:    "charges.get-unmasked-charge",
		OperationID: "getUnmaskedCharge",
		Method:      "GET",
		Path:        "/v1/charges/{id}/unmask",
		PathParams:  []string{"id"},
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
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	})
}

func newChargesGetUnmaskedChargeCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "charges.get-unmasked-charge",
		OperationID: "getUnmaskedCharge",
		Method:      "GET",
		Path:        "/v1/charges/{id}/unmask",
		PathParams:  []string{"id"},
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
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	}
	cmd := &cobra.Command{
		Use:     "get-unmasked-charge <id>",
		Short:   "Get an unmasked charge",
		Example: "  straddle charges get-unmasked-charge <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "charges.get-unmasked-charge",
			"straddle:operation-id": "getUnmaskedCharge",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/charges/{id}/unmask",
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
	applyOverlay("charges.get-unmasked-charge", cmd)
	return cmd
}
