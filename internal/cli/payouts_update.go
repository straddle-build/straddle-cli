// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("payouts.update", newPayoutsUpdateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "payouts.update",
		OperationID: "updatePayout",
		Method:      "PUT",
		Path:        "/v1/payouts/{id}",
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
				Name:        "amount",
				In:          surface.InBody,
				Key:         "/amount",
				Kind:        surface.KindInteger,
				Required:    true,
				Description: "Amount in cents.",
				Format:      "int32",
			},
			{
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Updated description for the payout.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Replacement metadata for the payout.",
			},
			{
				Name:        "payment-date",
				In:          surface.InBody,
				Key:         "/payment_date",
				Kind:        surface.KindString,
				Required:    true,
				Description: "New date for Straddle to submit the payout for processing.",
				Format:      "date",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newPayoutsUpdateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "payouts.update",
		OperationID: "updatePayout",
		Method:      "PUT",
		Path:        "/v1/payouts/{id}",
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
				Name:        "amount",
				In:          surface.InBody,
				Key:         "/amount",
				Kind:        surface.KindInteger,
				Required:    true,
				Description: "Amount in cents.",
				Format:      "int32",
			},
			{
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Updated description for the payout.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Replacement metadata for the payout.",
			},
			{
				Name:        "payment-date",
				In:          surface.InBody,
				Key:         "/payment_date",
				Kind:        surface.KindString,
				Required:    true,
				Description: "New date for Straddle to submit the payout for processing.",
				Format:      "date",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a payout",
		Example: "  straddle payouts update <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "payouts.update",
			"straddle:operation-id": "updatePayout",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/payouts/{id}",
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
	applyOverlay("payouts.update", cmd)
	return cmd
}
