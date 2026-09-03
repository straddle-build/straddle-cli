// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("linked-bank-accounts.cancel", newLinkedBankAccountsCancelGeneratedCmd)
	registerSurface(surface.Surface{
		Endpoint:    "linked-bank-accounts.cancel",
		OperationID: "cancelLinkedBankAccount",
		Method:      "PATCH",
		Path:        "/v1/linked_bank_accounts/{linked_bank_account_id}/cancel",
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
		},
		HasBody:              false,
		BodyRequired:         false,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newLinkedBankAccountsCancelGeneratedCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "linked-bank-accounts.cancel",
		OperationID: "cancelLinkedBankAccount",
		Method:      "PATCH",
		Path:        "/v1/linked_bank_accounts/{linked_bank_account_id}/cancel",
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
		},
		HasBody:              false,
		BodyRequired:         false,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "cancel <linked_bank_account_id>",
		Short:   "Cancel a linked bank account",
		Example: "  straddle linked-bank-accounts cancel <linked_bank_account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "linked-bank-accounts.cancel",
			"straddle:operation-id": "cancelLinkedBankAccount",
			"straddle:method":       "PATCH",
			"straddle:path":         "/v1/linked_bank_accounts/{linked_bank_account_id}/cancel",
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
	applyOverlay("linked-bank-accounts.cancel", cmd)
	return cmd
}
