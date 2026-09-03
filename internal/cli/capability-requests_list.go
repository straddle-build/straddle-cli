// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("capability-requests.list", newCapabilityRequestsListCmd)
	registerSurface(surface.Surface{
		Endpoint:    "capability-requests.list",
		OperationID: "listCapabilityRequests",
		Method:      "GET",
		Path:        "/v1/accounts/{account_id}/capability_requests",
		PathParams:  []string{"account_id"},
		Flags: []surface.Flag{
			{
				Name:        "category",
				In:          surface.InQuery,
				Key:         "category",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"payment_type", "customer_type", "consent_type"},
				Description: "Capability category to return.",
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
				Name:        "status",
				In:          surface.InQuery,
				Key:         "status",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"active", "inactive", "in_review", "rejected"},
				Description: "Capability request status to return.",
			},
			{
				Name:        "type",
				In:          surface.InQuery,
				Key:         "type",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"charges", "payouts", "individuals", "businesses", "signed_agreement", "internet"},
				Description: "Capability type to return.",
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

func newCapabilityRequestsListCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "capability-requests.list",
		OperationID: "listCapabilityRequests",
		Method:      "GET",
		Path:        "/v1/accounts/{account_id}/capability_requests",
		PathParams:  []string{"account_id"},
		Flags: []surface.Flag{
			{
				Name:        "category",
				In:          surface.InQuery,
				Key:         "category",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"payment_type", "customer_type", "consent_type"},
				Description: "Capability category to return.",
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
				Name:        "status",
				In:          surface.InQuery,
				Key:         "status",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"active", "inactive", "in_review", "rejected"},
				Description: "Capability request status to return.",
			},
			{
				Name:        "type",
				In:          surface.InQuery,
				Key:         "type",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"charges", "payouts", "individuals", "businesses", "signed_agreement", "internet"},
				Description: "Capability type to return.",
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
		Use:     "list <account_id>",
		Short:   "List capability requests",
		Example: "  straddle capability-requests list <account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "capability-requests.list",
			"straddle:operation-id": "listCapabilityRequests",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/accounts/{account_id}/capability_requests",
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
	applyOverlay("capability-requests.list", cmd)
	return cmd
}
