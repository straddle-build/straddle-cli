// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("charges.create", newChargesCreateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "charges.create",
		OperationID: "createCharge",
		Method:      "POST",
		Path:        "/v1/charges",
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
				Name:        "amount",
				In:          surface.InBody,
				Key:         "/amount",
				Kind:        surface.KindInteger,
				Required:    true,
				Description: "Amount in cents.",
				Format:      "int32",
			},
			{
				Name:        "config-auto-hold",
				In:          surface.InBody,
				Key:         "/config/auto_hold",
				Kind:        surface.KindBoolean,
				Description: "Whether to place the charge on hold automatically after creation.",
			},
			{
				Name:        "config-auto-hold-message",
				In:          surface.InBody,
				Key:         "/config/auto_hold_message",
				Kind:        surface.KindString,
				Description: "Reason for placing the charge on hold automatically.",
			},
			{
				Name:        "config-balance-check",
				In:          surface.InBody,
				Key:         "/config/balance_check",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"required", "enabled", "disabled"},
				Description: "Balance check mode to use before processing the charge.",
				Default:     "enabled",
			},
			{
				Name:        "config-sandbox-outcome",
				In:          surface.InBody,
				Key:         "/config/sandbox_outcome",
				Kind:        surface.KindString,
				Enum:        []string{"standard", "paid", "on_hold_daily_limit", "cancelled_for_fraud_risk", "cancelled_for_balance_check", "failed_insufficient_funds", "reversed_insufficient_funds", "failed_customer_dispute", "reversed_customer_dispute", "failed_closed_bank_account", "reversed_closed_bank_account", "failed_not_authorized", "reversed_not_authorized"},
				Description: "Payment will simulate processing if not Standard.",
			},
			{
				Name:        "consent-type",
				In:          surface.InBody,
				Key:         "/consent_type",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"internet", "signed"},
				Description: "How the customer authorized the charge.",
			},
			{
				Name:        "currency",
				In:          surface.InBody,
				Key:         "/currency",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Currency code.",
				Default:     "USD",
			},
			{
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Description shown on the customer's bank statement where supported.",
			},
			{
				Name:        "device-ip-address",
				In:          surface.InBody,
				Key:         "/device/ip_address",
				Kind:        surface.KindString,
				Required:    true,
				Description: "The IP address of the device used when the customer authorized the charge or payout.",
				Format:      "ipv4",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Your unique identifier for the charge.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined string key-value pairs.",
			},
			{
				Name:        "paykey",
				In:          surface.InBody,
				Key:         "/paykey",
				Kind:        surface.KindString,
				Required:    true,
				Description: "The paykey token that identifies the customer's bank account.",
			},
			{
				Name:        "payment-date",
				In:          surface.InBody,
				Key:         "/payment_date",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Date when Straddle submits the charge for processing.",
				Format:      "date",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newChargesCreateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "charges.create",
		OperationID: "createCharge",
		Method:      "POST",
		Path:        "/v1/charges",
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
				Name:        "amount",
				In:          surface.InBody,
				Key:         "/amount",
				Kind:        surface.KindInteger,
				Required:    true,
				Description: "Amount in cents.",
				Format:      "int32",
			},
			{
				Name:        "config-auto-hold",
				In:          surface.InBody,
				Key:         "/config/auto_hold",
				Kind:        surface.KindBoolean,
				Description: "Whether to place the charge on hold automatically after creation.",
			},
			{
				Name:        "config-auto-hold-message",
				In:          surface.InBody,
				Key:         "/config/auto_hold_message",
				Kind:        surface.KindString,
				Description: "Reason for placing the charge on hold automatically.",
			},
			{
				Name:        "config-balance-check",
				In:          surface.InBody,
				Key:         "/config/balance_check",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"required", "enabled", "disabled"},
				Description: "Balance check mode to use before processing the charge.",
				Default:     "enabled",
			},
			{
				Name:        "config-sandbox-outcome",
				In:          surface.InBody,
				Key:         "/config/sandbox_outcome",
				Kind:        surface.KindString,
				Enum:        []string{"standard", "paid", "on_hold_daily_limit", "cancelled_for_fraud_risk", "cancelled_for_balance_check", "failed_insufficient_funds", "reversed_insufficient_funds", "failed_customer_dispute", "reversed_customer_dispute", "failed_closed_bank_account", "reversed_closed_bank_account", "failed_not_authorized", "reversed_not_authorized"},
				Description: "Payment will simulate processing if not Standard.",
			},
			{
				Name:        "consent-type",
				In:          surface.InBody,
				Key:         "/consent_type",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"internet", "signed"},
				Description: "How the customer authorized the charge.",
			},
			{
				Name:        "currency",
				In:          surface.InBody,
				Key:         "/currency",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Currency code.",
				Default:     "USD",
			},
			{
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Description shown on the customer's bank statement where supported.",
			},
			{
				Name:        "device-ip-address",
				In:          surface.InBody,
				Key:         "/device/ip_address",
				Kind:        surface.KindString,
				Required:    true,
				Description: "The IP address of the device used when the customer authorized the charge or payout.",
				Format:      "ipv4",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Your unique identifier for the charge.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined string key-value pairs.",
			},
			{
				Name:        "paykey",
				In:          surface.InBody,
				Key:         "/paykey",
				Kind:        surface.KindString,
				Required:    true,
				Description: "The paykey token that identifies the customer's bank account.",
			},
			{
				Name:        "payment-date",
				In:          surface.InBody,
				Key:         "/payment_date",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Date when Straddle submits the charge for processing.",
				Format:      "date",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a charge",
		Example: "  straddle charges create",
		Annotations: map[string]string{
			"straddle:endpoint":     "charges.create",
			"straddle:operation-id": "createCharge",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/charges",
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
	applyOverlay("charges.create", cmd)
	return cmd
}
