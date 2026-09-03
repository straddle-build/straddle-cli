// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("accounts.onboard", newAccountsOnboardGeneratedCmd)
	registerSurface(surface.Surface{
		Endpoint:    "accounts.onboard",
		OperationID: "onboardAccount",
		Method:      "POST",
		Path:        "/v1/accounts/{account_id}/onboard",
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
				Name:        "terms-of-service-accepted-date",
				In:          surface.InBody,
				Key:         "/terms_of_service/accepted_date",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Date and time when the account accepted the Terms of Service.",
				Format:      "date-time",
			},
			{
				Name:        "terms-of-service-accepted-ip",
				In:          surface.InBody,
				Key:         "/terms_of_service/accepted_ip",
				Kind:        surface.KindString,
				Description: "IP address used to accept the Terms of Service.",
			},
			{
				Name:        "terms-of-service-accepted-user-agent",
				In:          surface.InBody,
				Key:         "/terms_of_service/accepted_user_agent",
				Kind:        surface.KindString,
				Description: "User agent of the browser or application that accepted the Terms of Service.",
			},
			{
				Name:        "terms-of-service-agreement-type",
				In:          surface.InBody,
				Key:         "/terms_of_service/agreement_type",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"embedded", "direct"},
				Description: "Agreement type.",
				Default:     "embedded",
			},
			{
				Name:        "terms-of-service-agreement-url",
				In:          surface.InBody,
				Key:         "/terms_of_service/agreement_url",
				Kind:        surface.KindString,
				Required:    true,
				Description: "URL of the accepted agreement.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newAccountsOnboardGeneratedCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "accounts.onboard",
		OperationID: "onboardAccount",
		Method:      "POST",
		Path:        "/v1/accounts/{account_id}/onboard",
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
				Name:        "terms-of-service-accepted-date",
				In:          surface.InBody,
				Key:         "/terms_of_service/accepted_date",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Date and time when the account accepted the Terms of Service.",
				Format:      "date-time",
			},
			{
				Name:        "terms-of-service-accepted-ip",
				In:          surface.InBody,
				Key:         "/terms_of_service/accepted_ip",
				Kind:        surface.KindString,
				Description: "IP address used to accept the Terms of Service.",
			},
			{
				Name:        "terms-of-service-accepted-user-agent",
				In:          surface.InBody,
				Key:         "/terms_of_service/accepted_user_agent",
				Kind:        surface.KindString,
				Description: "User agent of the browser or application that accepted the Terms of Service.",
			},
			{
				Name:        "terms-of-service-agreement-type",
				In:          surface.InBody,
				Key:         "/terms_of_service/agreement_type",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"embedded", "direct"},
				Description: "Agreement type.",
				Default:     "embedded",
			},
			{
				Name:        "terms-of-service-agreement-url",
				In:          surface.InBody,
				Key:         "/terms_of_service/agreement_url",
				Kind:        surface.KindString,
				Required:    true,
				Description: "URL of the accepted agreement.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "onboard <account_id>",
		Short:   "Onboard an account",
		Example: "  straddle accounts onboard <account_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "accounts.onboard",
			"straddle:operation-id": "onboardAccount",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/accounts/{account_id}/onboard",
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
	applyOverlay("accounts.onboard", cmd)
	return cmd
}
