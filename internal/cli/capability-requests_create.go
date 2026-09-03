// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("capability-requests.create", newCapabilityRequestsCreateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "capability-requests.create",
		OperationID: "createCapabilityRequest",
		Method:      "POST",
		Path:        "/v1/accounts/{account_id}/capability_requests",
		PathParams:  []string{"account_id"},
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
				Name:        "businesses-enable",
				In:          surface.InBody,
				Key:         "/businesses/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
			{
				Name:        "charges-daily-amount",
				In:          surface.InBody,
				Key:         "/charges/daily_amount",
				Kind:        surface.KindNumber,
				Description: "Daily charge amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "charges-enable",
				In:          surface.InBody,
				Key:         "/charges/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether to enable or disable charges for the account.",
			},
			{
				Name:        "charges-max-amount",
				In:          surface.InBody,
				Key:         "/charges/max_amount",
				Kind:        surface.KindNumber,
				Description: "Maximum amount in cents for one charge.",
				Format:      "double",
			},
			{
				Name:        "charges-monthly-amount",
				In:          surface.InBody,
				Key:         "/charges/monthly_amount",
				Kind:        surface.KindNumber,
				Description: "Monthly charge amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "charges-monthly-count",
				In:          surface.InBody,
				Key:         "/charges/monthly_count",
				Kind:        surface.KindInteger,
				Description: "Maximum number of charges per calendar month.",
				Format:      "int32",
			},
			{
				Name:        "individuals-enable",
				In:          surface.InBody,
				Key:         "/individuals/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
			{
				Name:        "internet-enable",
				In:          surface.InBody,
				Key:         "/internet/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
			{
				Name:        "payouts-daily-amount",
				In:          surface.InBody,
				Key:         "/payouts/daily_amount",
				Kind:        surface.KindNumber,
				Description: "Daily payout amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "payouts-enable",
				In:          surface.InBody,
				Key:         "/payouts/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether to enable or disable payouts for the account.",
			},
			{
				Name:        "payouts-max-amount",
				In:          surface.InBody,
				Key:         "/payouts/max_amount",
				Kind:        surface.KindNumber,
				Description: "Maximum amount in cents for one payout.",
				Format:      "double",
			},
			{
				Name:        "payouts-monthly-amount",
				In:          surface.InBody,
				Key:         "/payouts/monthly_amount",
				Kind:        surface.KindNumber,
				Description: "Monthly payout amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "payouts-monthly-count",
				In:          surface.InBody,
				Key:         "/payouts/monthly_count",
				Kind:        surface.KindInteger,
				Description: "Maximum number of payouts per calendar month.",
				Format:      "int32",
			},
			{
				Name:        "signed-agreement-enable",
				In:          surface.InBody,
				Key:         "/signed_agreement/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newCapabilityRequestsCreateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "capability-requests.create",
		OperationID: "createCapabilityRequest",
		Method:      "POST",
		Path:        "/v1/accounts/{account_id}/capability_requests",
		PathParams:  []string{"account_id"},
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
				Name:        "businesses-enable",
				In:          surface.InBody,
				Key:         "/businesses/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
			{
				Name:        "charges-daily-amount",
				In:          surface.InBody,
				Key:         "/charges/daily_amount",
				Kind:        surface.KindNumber,
				Description: "Daily charge amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "charges-enable",
				In:          surface.InBody,
				Key:         "/charges/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether to enable or disable charges for the account.",
			},
			{
				Name:        "charges-max-amount",
				In:          surface.InBody,
				Key:         "/charges/max_amount",
				Kind:        surface.KindNumber,
				Description: "Maximum amount in cents for one charge.",
				Format:      "double",
			},
			{
				Name:        "charges-monthly-amount",
				In:          surface.InBody,
				Key:         "/charges/monthly_amount",
				Kind:        surface.KindNumber,
				Description: "Monthly charge amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "charges-monthly-count",
				In:          surface.InBody,
				Key:         "/charges/monthly_count",
				Kind:        surface.KindInteger,
				Description: "Maximum number of charges per calendar month.",
				Format:      "int32",
			},
			{
				Name:        "individuals-enable",
				In:          surface.InBody,
				Key:         "/individuals/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
			{
				Name:        "internet-enable",
				In:          surface.InBody,
				Key:         "/internet/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
			{
				Name:        "payouts-daily-amount",
				In:          surface.InBody,
				Key:         "/payouts/daily_amount",
				Kind:        surface.KindNumber,
				Description: "Daily payout amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "payouts-enable",
				In:          surface.InBody,
				Key:         "/payouts/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether to enable or disable payouts for the account.",
			},
			{
				Name:        "payouts-max-amount",
				In:          surface.InBody,
				Key:         "/payouts/max_amount",
				Kind:        surface.KindNumber,
				Description: "Maximum amount in cents for one payout.",
				Format:      "double",
			},
			{
				Name:        "payouts-monthly-amount",
				In:          surface.InBody,
				Key:         "/payouts/monthly_amount",
				Kind:        surface.KindNumber,
				Description: "Monthly payout amount limit in cents.",
				Format:      "double",
			},
			{
				Name:        "payouts-monthly-count",
				In:          surface.InBody,
				Key:         "/payouts/monthly_count",
				Kind:        surface.KindInteger,
				Description: "Maximum number of payouts per calendar month.",
				Format:      "int32",
			},
			{
				Name:        "signed-agreement-enable",
				In:          surface.InBody,
				Key:         "/signed_agreement/enable",
				Kind:        surface.KindBoolean,
				Description: "Whether the request enables or disables the capability.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create <account_id>",
		Short:   "Create capability requests",
		Example: "  straddle capability-requests create <account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "capability-requests.create",
			"straddle:operation-id": "createCapabilityRequest",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/accounts/{account_id}/capability_requests",
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
	applyOverlay("capability-requests.create", cmd)
	return cmd
}
