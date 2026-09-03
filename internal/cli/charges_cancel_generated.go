// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("charges.cancel", newChargesCancelGeneratedCmd)
	registerSurface(surface.Surface{
		Endpoint:    "charges.cancel",
		OperationID: "cancelCharge",
		Method:      "PUT",
		Path:        "/v1/charges/{id}/cancel",
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
				Name:        "idempotency-key",
				In:          surface.InHeader,
				Key:         "Idempotency-Key",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated key for an idempotent request.",
			},
			{
				Name:        "request-id",
				In:          surface.InHeader,
				Key:         "Request-Id",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated identifier for tracing one request.",
			},
			{
				Name:        "reason",
				In:          surface.InBody,
				Key:         "/reason",
				Kind:        surface.KindString,
				Description: "Message explaining the charge status change.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newChargesCancelGeneratedCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "charges.cancel",
		OperationID: "cancelCharge",
		Method:      "PUT",
		Path:        "/v1/charges/{id}/cancel",
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
				Name:        "idempotency-key",
				In:          surface.InHeader,
				Key:         "Idempotency-Key",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated key for an idempotent request.",
			},
			{
				Name:        "request-id",
				In:          surface.InHeader,
				Key:         "Request-Id",
				Kind:        surface.KindString,
				Style:       surface.StyleSimple,
				Description: "Optional client-generated identifier for tracing one request.",
			},
			{
				Name:        "reason",
				In:          surface.InBody,
				Key:         "/reason",
				Kind:        surface.KindString,
				Description: "Message explaining the charge status change.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "cancel <id>",
		Short:   "Cancel a charge",
		Example: "  straddle charges cancel <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "charges.cancel",
			"straddle:operation-id": "cancelCharge",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/charges/{id}/cancel",
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
	applyOverlay("charges.cancel", cmd)
	return cmd
}
