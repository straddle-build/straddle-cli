// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("paykeys.set-paykey-verification-decision", newPaykeysSetPaykeyVerificationDecisionCmd)
	registerSurface(surface.Surface{
		Endpoint:    "paykeys.set-paykey-verification-decision",
		OperationID: "setPaykeyVerificationDecision",
		Method:      "PATCH",
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
				Name:     "status",
				In:       surface.InBody,
				Key:      "/status",
				Kind:     surface.KindString,
				Required: true,
				Enum:     []string{"active", "rejected"},
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newPaykeysSetPaykeyVerificationDecisionCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "paykeys.set-paykey-verification-decision",
		OperationID: "setPaykeyVerificationDecision",
		Method:      "PATCH",
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
				Name:     "status",
				In:       surface.InBody,
				Key:      "/status",
				Kind:     surface.KindString,
				Required: true,
				Enum:     []string{"active", "rejected"},
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "set-paykey-verification-decision <id>",
		Short:   "Set a paykey verification decision",
		Example: "  straddle paykeys set-paykey-verification-decision <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "paykeys.set-paykey-verification-decision",
			"straddle:operation-id": "setPaykeyVerificationDecision",
			"straddle:method":       "PATCH",
			"straddle:path":         "/v1/paykeys/{id}/review",
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
	applyOverlay("paykeys.set-paykey-verification-decision", cmd)
	return cmd
}
