// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("funding-events.simulate", newFundingEventsSimulateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "funding-events.simulate",
		OperationID: "simulateFundingEvent",
		Method:      "POST",
		Path:        "/v1/funding_events/simulate",
		PathParams:  []string{},
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
				Name:        "funding-event-job-type",
				In:          surface.InBody,
				Key:         "/funding_event_job_type",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"charges", "payouts"},
				Description: "Required.",
			},
			{
				Name:        "sandbox-outcome",
				In:          surface.InBody,
				Key:         "/sandbox_outcome",
				Kind:        surface.KindString,
				Enum:        []string{"standard", "paid", "on_hold_daily_limit", "cancelled_for_fraud_risk", "cancelled_for_balance_check", "failed_insufficient_funds", "reversed_insufficient_funds", "failed_customer_dispute", "reversed_customer_dispute", "failed_closed_bank_account", "reversed_closed_bank_account", "failed_not_authorized", "reversed_not_authorized"},
				Description: "Optional.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newFundingEventsSimulateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "funding-events.simulate",
		OperationID: "simulateFundingEvent",
		Method:      "POST",
		Path:        "/v1/funding_events/simulate",
		PathParams:  []string{},
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
				Name:        "funding-event-job-type",
				In:          surface.InBody,
				Key:         "/funding_event_job_type",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"charges", "payouts"},
				Description: "Required.",
			},
			{
				Name:        "sandbox-outcome",
				In:          surface.InBody,
				Key:         "/sandbox_outcome",
				Kind:        surface.KindString,
				Enum:        []string{"standard", "paid", "on_hold_daily_limit", "cancelled_for_fraud_risk", "cancelled_for_balance_check", "failed_insufficient_funds", "reversed_insufficient_funds", "failed_customer_dispute", "reversed_customer_dispute", "failed_closed_bank_account", "reversed_closed_bank_account", "failed_not_authorized", "reversed_not_authorized"},
				Description: "Optional.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "simulate",
		Short:   "Simulate a funding event",
		Example: "  straddle funding-events simulate",
		Annotations: map[string]string{
			"straddle:endpoint":     "funding-events.simulate",
			"straddle:operation-id": "simulateFundingEvent",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/funding_events/simulate",
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
	applyOverlay("funding-events.simulate", cmd)
	return cmd
}
