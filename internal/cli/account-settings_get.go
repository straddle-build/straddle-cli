// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("account-settings.get", newAccountSettingsGetCmd)
	registerSurface(surface.Surface{
		Endpoint:    "account-settings.get",
		OperationID: "getAccountSettings",
		Method:      "GET",
		Path:        "/v1/account_settings/{account_id}",
		PathParams:  []string{"account_id"},
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

func newAccountSettingsGetCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "account-settings.get",
		OperationID: "getAccountSettings",
		Method:      "GET",
		Path:        "/v1/account_settings/{account_id}",
		PathParams:  []string{"account_id"},
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
		Use:     "get <account_id>",
		Short:   "Get account settings",
		Example: "  straddle account-settings get <account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "account-settings.get",
			"straddle:operation-id": "getAccountSettings",
			"straddle:method":       "GET",
			"straddle:path":         "/v1/account_settings/{account_id}",
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
	applyOverlay("account-settings.get", cmd)
	return cmd
}
