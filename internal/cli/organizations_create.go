// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("organizations.create", newOrganizationsCreateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "organizations.create",
		OperationID: "createOrganization",
		Method:      "POST",
		Path:        "/v1/organizations",
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
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Your unique ID for the organization.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
			{
				Name:        "name",
				In:          surface.InBody,
				Key:         "/name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Organization name.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newOrganizationsCreateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "organizations.create",
		OperationID: "createOrganization",
		Method:      "POST",
		Path:        "/v1/organizations",
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
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Your unique ID for the organization.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
			{
				Name:        "name",
				In:          surface.InBody,
				Key:         "/name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Organization name.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an organization",
		Example: "  straddle organizations create",
		Annotations: map[string]string{
			"straddle:endpoint":     "organizations.create",
			"straddle:operation-id": "createOrganization",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/organizations",
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
	applyOverlay("organizations.create", cmd)
	return cmd
}
