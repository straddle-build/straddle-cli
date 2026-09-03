// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("linked-bank-accounts.create", newLinkedBankAccountsCreateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "linked-bank-accounts.create",
		OperationID: "createLinkedBankAccount",
		Method:      "POST",
		Path:        "/v1/linked_bank_accounts",
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
				Name:        "account-id",
				In:          surface.InBody,
				Key:         "/account_id",
				Kind:        surface.KindString,
				Description: "ID of the account that will own the linked bank account.",
				Format:      "uuid",
			},
			{
				Name:        "bank-account-account-holder",
				In:          surface.InBody,
				Key:         "/bank_account/account_holder",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Account holder name as it appears on the bank account.",
			},
			{
				Name:        "bank-account-account-number",
				In:          surface.InBody,
				Key:         "/bank_account/account_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "The bank account number.",
			},
			{
				Name:        "bank-account-routing-number",
				In:          surface.InBody,
				Key:         "/bank_account/routing_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Nine-digit ABA routing number.",
			},
			{
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Description: "Your description for the linked bank account.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
			{
				Name:        "platform-id",
				In:          surface.InBody,
				Key:         "/platform_id",
				Kind:        surface.KindString,
				Description: "ID of the platform to associate with the linked bank account.",
				Format:      "uuid",
			},
			{
				Name:        "purposes",
				In:          surface.InBody,
				Key:         "/purposes",
				Kind:        surface.KindString,
				Array:       true,
				Enum:        []string{"charges", "payouts", "billing"},
				Description: "Payment purposes for the linked bank account.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newLinkedBankAccountsCreateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "linked-bank-accounts.create",
		OperationID: "createLinkedBankAccount",
		Method:      "POST",
		Path:        "/v1/linked_bank_accounts",
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
				Name:        "account-id",
				In:          surface.InBody,
				Key:         "/account_id",
				Kind:        surface.KindString,
				Description: "ID of the account that will own the linked bank account.",
				Format:      "uuid",
			},
			{
				Name:        "bank-account-account-holder",
				In:          surface.InBody,
				Key:         "/bank_account/account_holder",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Account holder name as it appears on the bank account.",
			},
			{
				Name:        "bank-account-account-number",
				In:          surface.InBody,
				Key:         "/bank_account/account_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "The bank account number.",
			},
			{
				Name:        "bank-account-routing-number",
				In:          surface.InBody,
				Key:         "/bank_account/routing_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Nine-digit ABA routing number.",
			},
			{
				Name:        "description",
				In:          surface.InBody,
				Key:         "/description",
				Kind:        surface.KindString,
				Description: "Your description for the linked bank account.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
			{
				Name:        "platform-id",
				In:          surface.InBody,
				Key:         "/platform_id",
				Kind:        surface.KindString,
				Description: "ID of the platform to associate with the linked bank account.",
				Format:      "uuid",
			},
			{
				Name:        "purposes",
				In:          surface.InBody,
				Key:         "/purposes",
				Kind:        surface.KindString,
				Array:       true,
				Enum:        []string{"charges", "payouts", "billing"},
				Description: "Payment purposes for the linked bank account.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a linked bank account",
		Example: "  straddle linked-bank-accounts create",
		Annotations: map[string]string{
			"straddle:endpoint":     "linked-bank-accounts.create",
			"straddle:operation-id": "createLinkedBankAccount",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/linked_bank_accounts",
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
	applyOverlay("linked-bank-accounts.create", cmd)
	return cmd
}
