// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("accounts.simulate-account-onboarding", newAccountsSimulateAccountOnboardingCmd)
	registerSurface(surface.Surface{
		Endpoint:    "accounts.simulate-account-onboarding",
		OperationID: "simulateAccountOnboarding",
		Method:      "POST",
		Path:        "/v1/accounts/{account_id}/simulate",
		PathParams:  []string{"account_id"},
		Flags: []surface.Flag{
			{
				Name:        "final-status",
				In:          surface.InQuery,
				Key:         "final_status",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"onboarding", "active"},
				Description: "Final account status to produce in the sandbox simulation.",
			},
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
		},
		HasBody:              false,
		BodyRequired:         false,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newAccountsSimulateAccountOnboardingCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "accounts.simulate-account-onboarding",
		OperationID: "simulateAccountOnboarding",
		Method:      "POST",
		Path:        "/v1/accounts/{account_id}/simulate",
		PathParams:  []string{"account_id"},
		Flags: []surface.Flag{
			{
				Name:        "final-status",
				In:          surface.InQuery,
				Key:         "final_status",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"onboarding", "active"},
				Description: "Final account status to produce in the sandbox simulation.",
			},
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
		},
		HasBody:              false,
		BodyRequired:         false,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "simulate-account-onboarding <account_id>",
		Short:   "Simulate status transitions for a sandbox account",
		Example: "  straddle accounts simulate-account-onboarding <account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "accounts.simulate-account-onboarding",
			"straddle:operation-id": "simulateAccountOnboarding",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/accounts/{account_id}/simulate",
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
	applyOverlay("accounts.simulate-account-onboarding", cmd)
	return cmd
}
