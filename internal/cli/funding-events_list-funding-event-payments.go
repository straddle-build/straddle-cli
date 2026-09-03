// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("funding-events.list-funding-event-payments", newFundingEventsListFundingEventPaymentsCmd)
	registerSurface(surface.Surface{
		Endpoint:    "funding-events.list-funding-event-payments",
		OperationID: "listFundingEventPayments",
		Method:      "GET",
		Path:        "/v1/funding_event_payments/{id}",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "default-page-size",
				In:          surface.InQuery,
				Key:         "default_page_size",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Default number of results returned per page.",
				Format:      "int32",
			},
			{
				Name:        "default-sort",
				In:          surface.InQuery,
				Key:         "default_sort",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"created_at", "payment_date", "effective_at", "id"},
				Description: "Default field used to sort the results.",
			},
			{
				Name:        "default-sort-order",
				In:          surface.InQuery,
				Key:         "default_sort_order",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"asc", "desc"},
				Description: "Default order in which to sort the results.",
				Default:     "asc",
			},
			{
				Name:        "include-metadata",
				In:          surface.InQuery,
				Key:         "include_metadata",
				Kind:        surface.KindBoolean,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "When `true`, includes each payment's metadata.",
			},
			{
				Name:        "page-number",
				In:          surface.InQuery,
				Key:         "page_number",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Results page number.",
				Format:      "int32",
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
			},
			{
				Name:        "sort-by",
				In:          surface.InQuery,
				Key:         "sort_by",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"created_at", "payment_date", "effective_at", "id"},
				Description: "Field used to sort the results.",
			},
			{
				Name:        "sort-order",
				In:          surface.InQuery,
				Key:         "sort_order",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"asc", "desc"},
				Description: "Order in which to sort the results.",
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
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	})
}

func newFundingEventsListFundingEventPaymentsCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "funding-events.list-funding-event-payments",
		OperationID: "listFundingEventPayments",
		Method:      "GET",
		Path:        "/v1/funding_event_payments/{id}",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "default-page-size",
				In:          surface.InQuery,
				Key:         "default_page_size",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Default number of results returned per page.",
				Format:      "int32",
			},
			{
				Name:        "default-sort",
				In:          surface.InQuery,
				Key:         "default_sort",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"created_at", "payment_date", "effective_at", "id"},
				Description: "Default field used to sort the results.",
			},
			{
				Name:        "default-sort-order",
				In:          surface.InQuery,
				Key:         "default_sort_order",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"asc", "desc"},
				Description: "Default order in which to sort the results.",
				Default:     "asc",
			},
			{
				Name:        "include-metadata",
				In:          surface.InQuery,
				Key:         "include_metadata",
				Kind:        surface.KindBoolean,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "When `true`, includes each payment's metadata.",
			},
			{
				Name:        "page-number",
				In:          surface.InQuery,
				Key:         "page_number",
				Kind:        surface.KindInteger,
				Style:       surface.StyleForm,
				Explode:     true,
				Description: "Results page number.",
				Format:      "int32",
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
			},
			{
				Name:        "sort-by",
				In:          surface.InQuery,
				Key:         "sort_by",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"created_at", "payment_date", "effective_at", "id"},
				Description: "Field used to sort the results.",
			},
			{
				Name:        "sort-order",
				In:          surface.InQuery,
				Key:         "sort_order",
				Kind:        surface.KindString,
				Style:       surface.StyleForm,
				Explode:     true,
				Enum:        []string{"asc", "desc"},
				Description: "Order in which to sort the results.",
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
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	}
	cmd := &cobra.Command{
		Use:     "list-funding-event-payments <id>",
		Short:   "List funding event payments",
		Example: "  straddle funding-events list-funding-event-payments <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "funding-events.list-funding-event-payments",
			"straddle:operation-id": "listFundingEventPayments",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/funding_event_payments/{id}",
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
	applyOverlay("funding-events.list-funding-event-payments", cmd)
	return cmd
}
