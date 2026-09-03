// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("paykeys.refresh-paykey-review", newPaykeysRefreshPaykeyReviewCmd)
	registerSurface(surface.Surface{
		Endpoint:    "paykeys.refresh-paykey-review",
		OperationID: "refreshPaykeyReview",
		Method:      "PUT",
		Path:        "/v1/paykeys/{id}/refresh_review",
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
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newPaykeysRefreshPaykeyReviewCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "paykeys.refresh-paykey-review",
		OperationID: "refreshPaykeyReview",
		Method:      "PUT",
		Path:        "/v1/paykeys/{id}/refresh_review",
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
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "refresh-paykey-review <id>",
		Short:   "Refresh a paykey review",
		Example: "  straddle paykeys refresh-paykey-review <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "paykeys.refresh-paykey-review",
			"straddle:operation-id": "refreshPaykeyReview",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/paykeys/{id}/refresh_review",
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
	applyOverlay("paykeys.refresh-paykey-review", cmd)
	return cmd
}
