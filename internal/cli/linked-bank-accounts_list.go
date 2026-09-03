// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("linked-bank-accounts.list", newLinkedBankAccountsListCmd)
	registerSurface(surface.Surface{
		Endpoint:    "linked-bank-accounts.list",
		OperationID: "listLinkedBankAccounts",
		Method:      "GET",
		Path:        "/v1/linked_bank_accounts",
		PathParams:  []string{},
		Flags: []surface.Flag{
			{
				Name:        "account-id",
				In:          surface.InQuery,
				Key:         "account_id",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Account ID used to filter the results.",
				Format:      "uuid",
			},
			{
				Name:        "level",
				In:          surface.InQuery,
				Key:         "level",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"account", "platform"},
				Description: "Scope of linked bank accounts to return.",
			},
			{
				Name:        "page-number",
				In:          surface.InQuery,
				Key:         "page_number",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Page number.",
				Format:      "int32",
				Default:     "1",
			},
			{
				Name:        "page-size",
				In:          surface.InQuery,
				Key:         "page_size",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Number of results per page.",
				Format:      "int32",
				Default:     "100",
			},
			{
				Name:        "purpose",
				In:          surface.InQuery,
				Key:         "purpose",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"charges", "payouts", "billing"},
				Description: "Linked bank account purpose.",
			},
			{
				Name:        "sort-by",
				In:          surface.InQuery,
				Key:         "sort_by",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Field used to sort results.",
				Default:     "id",
			},
			{
				Name:        "sort-order",
				In:          surface.InQuery,
				Key:         "sort_order",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"asc", "desc"},
				Description: "Sort direction.",
				Default:     "asc",
			},
			{
				Name:        "status",
				In:          surface.InQuery,
				Key:         "status",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"created", "onboarding", "active", "rejected", "inactive", "canceled"},
				Description: "Linked bank account status.",
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

func newLinkedBankAccountsListCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "linked-bank-accounts.list",
		OperationID: "listLinkedBankAccounts",
		Method:      "GET",
		Path:        "/v1/linked_bank_accounts",
		PathParams:  []string{},
		Flags: []surface.Flag{
			{
				Name:        "account-id",
				In:          surface.InQuery,
				Key:         "account_id",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Account ID used to filter the results.",
				Format:      "uuid",
			},
			{
				Name:        "level",
				In:          surface.InQuery,
				Key:         "level",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"account", "platform"},
				Description: "Scope of linked bank accounts to return.",
			},
			{
				Name:        "page-number",
				In:          surface.InQuery,
				Key:         "page_number",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Page number.",
				Format:      "int32",
				Default:     "1",
			},
			{
				Name:        "page-size",
				In:          surface.InQuery,
				Key:         "page_size",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Number of results per page.",
				Format:      "int32",
				Default:     "100",
			},
			{
				Name:        "purpose",
				In:          surface.InQuery,
				Key:         "purpose",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"charges", "payouts", "billing"},
				Description: "Linked bank account purpose.",
			},
			{
				Name:        "sort-by",
				In:          surface.InQuery,
				Key:         "sort_by",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Field used to sort results.",
				Default:     "id",
			},
			{
				Name:        "sort-order",
				In:          surface.InQuery,
				Key:         "sort_order",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"asc", "desc"},
				Description: "Sort direction.",
				Default:     "asc",
			},
			{
				Name:        "status",
				In:          surface.InQuery,
				Key:         "status",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"created", "onboarding", "active", "rejected", "inactive", "canceled"},
				Description: "Linked bank account status.",
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
		Use:     "list",
		Short:   "List linked bank accounts",
		Example: "  straddle linked-bank-accounts list",
		Annotations: map[string]string{
			"straddle:endpoint":     "linked-bank-accounts.list",
			"straddle:operation-id": "listLinkedBankAccounts",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/linked_bank_accounts",
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
	applyOverlay("linked-bank-accounts.list", cmd)
	return cmd
}
