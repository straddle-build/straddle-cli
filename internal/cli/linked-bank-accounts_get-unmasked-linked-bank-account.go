// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("linked-bank-accounts.get-unmasked-linked-bank-account", newLinkedBankAccountsGetUnmaskedLinkedBankAccountCmd)
	registerSurface(surface.Surface{
		Endpoint:    "linked-bank-accounts.get-unmasked-linked-bank-account",
		OperationID: "getUnmaskedLinkedBankAccount",
		Method:      "GET",
		Path:        "/v1/linked_bank_accounts/{linked_bank_account_id}/unmask",
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
		ReadOnly:             true,
	})
}

func newLinkedBankAccountsGetUnmaskedLinkedBankAccountCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "linked-bank-accounts.get-unmasked-linked-bank-account",
		OperationID: "getUnmaskedLinkedBankAccount",
		Method:      "GET",
		Path:        "/v1/linked_bank_accounts/{linked_bank_account_id}/unmask",
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
		ReadOnly:             true,
	}
	cmd := &cobra.Command{
		Use:     "get-unmasked-linked-bank-account <linked_bank_account_id>",
		Short:   "Get an unmasked linked bank account",
		Example: "  straddle linked-bank-accounts get-unmasked-linked-bank-account <linked_bank_account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "linked-bank-accounts.get-unmasked-linked-bank-account",
			"straddle:operation-id": "getUnmaskedLinkedBankAccount",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/linked_bank_accounts/{linked_bank_account_id}/unmask",
			"mcp:read-only":         "true",
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
	applyOverlay("linked-bank-accounts.get-unmasked-linked-bank-account", cmd)
	return cmd
}
