// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("paykeys.unblock-paykey", newPaykeysUnblockPaykeyCmd)
	registerSurface(surface.Surface{
		Endpoint:    "paykeys.unblock-paykey",
		OperationID: "unblockPaykey",
		Method:      "PATCH",
		Path:        "/v1/paykeys/{id}/unblock",
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
			{
				Name:        "message",
				In:          surface.InBody,
				Key:         "/message",
				Kind:        surface.KindString,
				Description: "Optional message describing the reason for unblocking.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newPaykeysUnblockPaykeyCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "paykeys.unblock-paykey",
		OperationID: "unblockPaykey",
		Method:      "PATCH",
		Path:        "/v1/paykeys/{id}/unblock",
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
			{
				Name:        "message",
				In:          surface.InBody,
				Key:         "/message",
				Kind:        surface.KindString,
				Description: "Optional message describing the reason for unblocking.",
			},
		},
		HasBody:              true,
		BodyRequired:         false,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "unblock-paykey <id>",
		Short:   "Unblock a paykey",
		Example: "  straddle paykeys unblock-paykey <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "paykeys.unblock-paykey",
			"straddle:operation-id": "unblockPaykey",
			"straddle:method":       "PATCH",
			"straddle:path":         "/v1/paykeys/{id}/unblock",
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
	applyOverlay("paykeys.unblock-paykey", cmd)
	return cmd
}
