// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("linked-bank-accounts.update", newLinkedBankAccountsUpdateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "linked-bank-accounts.update",
		OperationID: "updateLinkedBankAccount",
		Method:      "PUT",
		Path:        "/v1/linked_bank_accounts/{linked_bank_account_id}",
		PathParams:  []string{"linked_bank_account_id"},
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
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newLinkedBankAccountsUpdateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "linked-bank-accounts.update",
		OperationID: "updateLinkedBankAccount",
		Method:      "PUT",
		Path:        "/v1/linked_bank_accounts/{linked_bank_account_id}",
		PathParams:  []string{"linked_bank_account_id"},
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
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "update <linked_bank_account_id>",
		Short:   "Update a linked bank account",
		Example: "  straddle linked-bank-accounts update <linked_bank_account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "linked-bank-accounts.update",
			"straddle:operation-id": "updateLinkedBankAccount",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/linked_bank_accounts/{linked_bank_account_id}",
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
	applyOverlay("linked-bank-accounts.update", cmd)
	return cmd
}
