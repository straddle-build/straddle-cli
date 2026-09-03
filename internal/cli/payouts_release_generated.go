// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("payouts.release", newPayoutsReleaseGeneratedCmd)
	registerSurface(surface.Surface{
		Endpoint:    "payouts.release",
		OperationID: "releasePayout",
		Method:      "PUT",
		Path:        "/v1/payouts/{id}/release",
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
				Description: "Message explaining the payout status change.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newPayoutsReleaseGeneratedCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "payouts.release",
		OperationID: "releasePayout",
		Method:      "PUT",
		Path:        "/v1/payouts/{id}/release",
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
				Description: "Message explaining the payout status change.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "release <id>",
		Short:   "Release a payout",
		Example: "  straddle payouts release <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "payouts.release",
			"straddle:operation-id": "releasePayout",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/payouts/{id}/release",
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
	applyOverlay("payouts.release", cmd)
	return cmd
}
