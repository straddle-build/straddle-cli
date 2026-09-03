// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("bridge.create-bank-account-paykey", newBridgeCreateBankAccountPaykeyCmd)
	registerSurface(surface.Surface{
		Endpoint:    "bridge.create-bank-account-paykey",
		OperationID: "createBankAccountPaykey",
		Method:      "POST",
		Path:        "/v1/bridge/bank_account",
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
				Name:        "account-number",
				In:          surface.InBody,
				Key:         "/account_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Bank account number.",
			},
			{
				Name:     "account-type",
				In:       surface.InBody,
				Key:      "/account_type",
				Kind:     surface.KindString,
				Required: true,
				Enum:     []string{"checking", "savings"},
			},
			{
				Name: "config-processing-method",
				In:   surface.InBody,
				Key:  "/config/processing_method",
				Kind: surface.KindString,
				Enum: []string{"inline", "background", "skip"},
			},
			{
				Name: "config-sandbox-outcome",
				In:   surface.InBody,
				Key:  "/config/sandbox_outcome",
				Kind: surface.KindString,
				Enum: []string{"standard", "active", "rejected", "review"},
			},
			{
				Name:        "customer-id",
				In:          surface.InBody,
				Key:         "/customer_id",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Unique identifier for the customer associated with the paykey.",
				Format:      "uuid",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Unique identifier for the paykey in your system.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs associated with the paykey.",
			},
			{
				Name:        "routing-number",
				In:          surface.InBody,
				Key:         "/routing_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Bank routing number.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newBridgeCreateBankAccountPaykeyCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "bridge.create-bank-account-paykey",
		OperationID: "createBankAccountPaykey",
		Method:      "POST",
		Path:        "/v1/bridge/bank_account",
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
				Name:        "account-number",
				In:          surface.InBody,
				Key:         "/account_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Bank account number.",
			},
			{
				Name:     "account-type",
				In:       surface.InBody,
				Key:      "/account_type",
				Kind:     surface.KindString,
				Required: true,
				Enum:     []string{"checking", "savings"},
			},
			{
				Name: "config-processing-method",
				In:   surface.InBody,
				Key:  "/config/processing_method",
				Kind: surface.KindString,
				Enum: []string{"inline", "background", "skip"},
			},
			{
				Name: "config-sandbox-outcome",
				In:   surface.InBody,
				Key:  "/config/sandbox_outcome",
				Kind: surface.KindString,
				Enum: []string{"standard", "active", "rejected", "review"},
			},
			{
				Name:        "customer-id",
				In:          surface.InBody,
				Key:         "/customer_id",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Unique identifier for the customer associated with the paykey.",
				Format:      "uuid",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Unique identifier for the paykey in your system.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs associated with the paykey.",
			},
			{
				Name:        "routing-number",
				In:          surface.InBody,
				Key:         "/routing_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Bank routing number.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create-bank-account-paykey",
		Short:   "Create a paykey from bank account details",
		Example: "  straddle bridge create-bank-account-paykey",
		Annotations: map[string]string{
			"straddle:endpoint":     "bridge.create-bank-account-paykey",
			"straddle:operation-id": "createBankAccountPaykey",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/bridge/bank_account",
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
	applyOverlay("bridge.create-bank-account-paykey", cmd)
	return cmd
}
