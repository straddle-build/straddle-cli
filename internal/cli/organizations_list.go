// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("organizations.list", newOrganizationsListCmd)
	registerSurface(surface.Surface{
		Endpoint:    "organizations.list",
		OperationID: "listOrganizations",
		Method:      "GET",
		Path:        "/v1/organizations",
		PathParams:  []string{},
		Flags: []surface.Flag{
			{
				Name:        "external-id",
				In:          surface.InQuery,
				Key:         "external_id",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Your external ID for the organization.",
			},
			{
				Name:        "name",
				In:          surface.InQuery,
				Key:         "name",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Organization name.",
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

func newOrganizationsListCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "organizations.list",
		OperationID: "listOrganizations",
		Method:      "GET",
		Path:        "/v1/organizations",
		PathParams:  []string{},
		Flags: []surface.Flag{
			{
				Name:        "external-id",
				In:          surface.InQuery,
				Key:         "external_id",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Your external ID for the organization.",
			},
			{
				Name:        "name",
				In:          surface.InQuery,
				Key:         "name",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Organization name.",
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
		Short:   "List organizations",
		Example: "  straddle organizations list",
		Annotations: map[string]string{
			"straddle:endpoint":     "organizations.list",
			"straddle:operation-id": "listOrganizations",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/organizations",
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
	applyOverlay("organizations.list", cmd)
	return cmd
}
