// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("bridge.create-bridge-token", newBridgeCreateBridgeTokenCmd)
	registerSurface(surface.Surface{
		Endpoint:    "bridge.create-bridge-token",
		OperationID: "createBridgeToken",
		Method:      "POST",
		Path:        "/v1/bridge/initialize",
		PathParams:  []string{},
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
				Name: "config-processing-method",
				In:   surface.InBody,
				Key:  "/config/processing_method",
				Kind: surface.KindString,
				Enum: []string{"inline", "background", "skip"},
			},
			{
				Name: "config-sandbox-outcome",
				In:   surface.InBody,
				Key:  "/config/sandbox_outcome",
				Kind: surface.KindString,
				Enum: []string{"standard", "active", "rejected", "review"},
			},
			{
				Name:        "customer-id",
				In:          surface.InBody,
				Key:         "/customer_id",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Unique identifier for the customer associated with the Bridge session.",
				Format:      "uuid",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Unique identifier for the paykey in your system.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newBridgeCreateBridgeTokenCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "bridge.create-bridge-token",
		OperationID: "createBridgeToken",
		Method:      "POST",
		Path:        "/v1/bridge/initialize",
		PathParams:  []string{},
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
				Name: "config-processing-method",
				In:   surface.InBody,
				Key:  "/config/processing_method",
				Kind: surface.KindString,
				Enum: []string{"inline", "background", "skip"},
			},
			{
				Name: "config-sandbox-outcome",
				In:   surface.InBody,
				Key:  "/config/sandbox_outcome",
				Kind: surface.KindString,
				Enum: []string{"standard", "active", "rejected", "review"},
			},
			{
				Name:        "customer-id",
				In:          surface.InBody,
				Key:         "/customer_id",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Unique identifier for the customer associated with the Bridge session.",
				Format:      "uuid",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Unique identifier for the paykey in your system.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create-bridge-token",
		Short:   "Create a Bridge widget session token",
		Example: "  straddle bridge create-bridge-token",
		Annotations: map[string]string{
			"straddle:endpoint":     "bridge.create-bridge-token",
			"straddle:operation-id": "createBridgeToken",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/bridge/initialize",
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
	applyOverlay("bridge.create-bridge-token", cmd)
	return cmd
}
