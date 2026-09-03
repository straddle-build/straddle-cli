// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("paykeys.get-paykey-review", newPaykeysGetPaykeyReviewCmd)
	registerSurface(surface.Surface{
		Endpoint:    "paykeys.get-paykey-review",
		OperationID: "getPaykeyReview",
		Method:      "GET",
		Path:        "/v1/paykeys/{id}/review",
		PathParams:  []string{"id"},
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
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	})
}

func newPaykeysGetPaykeyReviewCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "paykeys.get-paykey-review",
		OperationID: "getPaykeyReview",
		Method:      "GET",
		Path:        "/v1/paykeys/{id}/review",
		PathParams:  []string{"id"},
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
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	}
	cmd := &cobra.Command{
		Use:     "get-paykey-review <id>",
		Short:   "Get a paykey review",
		Example: "  straddle paykeys get-paykey-review <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "paykeys.get-paykey-review",
			"straddle:operation-id": "getPaykeyReview",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/paykeys/{id}/review",
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
	applyOverlay("paykeys.get-paykey-review", cmd)
	return cmd
}
