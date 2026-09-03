// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("paykeys.cancel", newPaykeysCancelGeneratedCmd)
	registerSurface(surface.Surface{
		Endpoint:    "paykeys.cancel",
		OperationID: "cancelPaykey",
		Method:      "PUT",
		Path:        "/v1/paykeys/{id}/cancel",
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
				Description: "Reason for canceling the paykey.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newPaykeysCancelGeneratedCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "paykeys.cancel",
		OperationID: "cancelPaykey",
		Method:      "PUT",
		Path:        "/v1/paykeys/{id}/cancel",
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
				Description: "Reason for canceling the paykey.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "cancel <id>",
		Short:   "Cancel a paykey",
		Example: "  straddle paykeys cancel <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "paykeys.cancel",
			"straddle:operation-id": "cancelPaykey",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/paykeys/{id}/cancel",
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
	applyOverlay("paykeys.cancel", cmd)
	return cmd
}
