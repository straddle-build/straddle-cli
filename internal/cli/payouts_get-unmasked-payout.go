// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("payouts.get-unmasked-payout", newPayoutsGetUnmaskedPayoutCmd)
	registerSurface(surface.Surface{
		Endpoint:    "payouts.get-unmasked-payout",
		OperationID: "getUnmaskedPayout",
		Method:      "GET",
		Path:        "/v1/payouts/{id}/unmask",
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

func newPayoutsGetUnmaskedPayoutCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "payouts.get-unmasked-payout",
		OperationID: "getUnmaskedPayout",
		Method:      "GET",
		Path:        "/v1/payouts/{id}/unmask",
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
		Use:     "get-unmasked-payout <id>",
		Short:   "Get an unmasked payout",
		Example: "  straddle payouts get-unmasked-payout <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "payouts.get-unmasked-payout",
			"straddle:operation-id": "getUnmaskedPayout",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/payouts/{id}/unmask",
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
	applyOverlay("payouts.get-unmasked-payout", cmd)
	return cmd
}
