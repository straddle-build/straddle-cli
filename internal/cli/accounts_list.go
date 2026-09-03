// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("accounts.list", newAccountsListCmd)
	registerSurface(surface.Surface{
		Endpoint:    "accounts.list",
		OperationID: "listAccounts",
		Method:      "GET",
		Path:        "/v1/accounts",
		PathParams:  []string{},
		Flags: []surface.Flag{
			{
				Name:        "external-id",
				In:          surface.InQuery,
				Key:         "external_id",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Your external ID for the account.",
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
				Name:        "search-text",
				In:          surface.InQuery,
				Key:         "search_text",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Text to search for across account fields.",
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
				Enum:        []string{"created", "onboarding", "active", "rejected", "inactive"},
				Description: "Account status to return.",
			},
			{
				Name:        "type",
				In:          surface.InQuery,
				Key:         "type",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"business"},
				Description: "Account type to return.",
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

func newAccountsListCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "accounts.list",
		OperationID: "listAccounts",
		Method:      "GET",
		Path:        "/v1/accounts",
		PathParams:  []string{},
		Flags: []surface.Flag{
			{
				Name:        "external-id",
				In:          surface.InQuery,
				Key:         "external_id",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Your external ID for the account.",
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
				Name:        "search-text",
				In:          surface.InQuery,
				Key:         "search_text",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Text to search for across account fields.",
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
				Enum:        []string{"created", "onboarding", "active", "rejected", "inactive"},
				Description: "Account status to return.",
			},
			{
				Name:        "type",
				In:          surface.InQuery,
				Key:         "type",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"business"},
				Description: "Account type to return.",
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
		Short:   "List accounts",
		Example: "  straddle accounts list",
		Annotations: map[string]string{
			"straddle:endpoint":     "accounts.list",
			"straddle:operation-id": "listAccounts",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/accounts",
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
	applyOverlay("accounts.list", cmd)
	return cmd
}
