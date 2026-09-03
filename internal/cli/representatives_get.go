// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("representatives.get", newRepresentativesGetCmd)
	registerSurface(surface.Surface{
		Endpoint:    "representatives.get",
		OperationID: "getRepresentative",
		Method:      "GET",
		Path:        "/v1/representatives/{representative_id}",
		PathParams:  []string{"representative_id"},
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
		AcceptsAccountHeader: false,
		ReadOnly:             true,
	})
}

func newRepresentativesGetCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "representatives.get",
		OperationID: "getRepresentative",
		Method:      "GET",
		Path:        "/v1/representatives/{representative_id}",
		PathParams:  []string{"representative_id"},
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
		AcceptsAccountHeader: false,
		ReadOnly:             true,
	}
	cmd := &cobra.Command{
		Use:     "get <representative_id>",
		Short:   "Get a representative",
		Example: "  straddle representatives get <representative_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "representatives.get",
			"straddle:operation-id": "getRepresentative",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/representatives/{representative_id}",
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
	applyOverlay("representatives.get", cmd)
	return cmd
}
