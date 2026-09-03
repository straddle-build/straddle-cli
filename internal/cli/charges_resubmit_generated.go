// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("charges.resubmit", newChargesResubmitGeneratedCmd)
	registerSurface(surface.Surface{
		Endpoint:    "charges.resubmit",
		OperationID: "resubmitCharge",
		Method:      "POST",
		Path:        "/v1/charges/{id}/resubmit",
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
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Description: "Description for the resubmitted charge.",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Your unique identifier for the resubmitted charge.",
			},
			{
				Name:        "payment-date",
				In:          surface.InBody,
				Key:         "/payment_date",
				Kind:        surface.KindString,
				Description: "Date when Straddle submits the resubmitted charge for processing.",
				Format:      "date",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newChargesResubmitGeneratedCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "charges.resubmit",
		OperationID: "resubmitCharge",
		Method:      "POST",
		Path:        "/v1/charges/{id}/resubmit",
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
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Description: "Description for the resubmitted charge.",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Your unique identifier for the resubmitted charge.",
			},
			{
				Name:        "payment-date",
				In:          surface.InBody,
				Key:         "/payment_date",
				Kind:        surface.KindString,
				Description: "Date when Straddle submits the resubmitted charge for processing.",
				Format:      "date",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "resubmit <id>",
		Short:   "Resubmit a charge",
		Example: "  straddle charges resubmit <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "charges.resubmit",
			"straddle:operation-id": "resubmitCharge",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/charges/{id}/resubmit",
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
	applyOverlay("charges.resubmit", cmd)
	return cmd
}
