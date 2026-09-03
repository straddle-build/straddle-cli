// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("representatives.update", newRepresentativesUpdateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "representatives.update",
		OperationID: "updateRepresentative",
		Method:      "PUT",
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
				Name:        "dob",
				In:          surface.InBody,
				Key:         "/dob",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's date of birth in `YYYY-MM-DD` format.",
				Format:      "date",
			},
			{
				Name:        "email",
				In:          surface.InBody,
				Key:         "/email",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's email address.",
				Format:      "email",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Your unique ID for the representative.",
			},
			{
				Name:        "first-name",
				In:          surface.InBody,
				Key:         "/first_name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's first name.",
			},
			{
				Name:        "last-name",
				In:          surface.InBody,
				Key:         "/last_name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's last name.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
			{
				Name:        "mobile-number",
				In:          surface.InBody,
				Key:         "/mobile_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's mobile phone number in E.164 format.",
			},
			{
				Name:        "relationship-control",
				In:          surface.InBody,
				Key:         "/relationship/control",
				Kind:        surface.KindBoolean,
				Required:    true,
				Description: "Whether the representative controls, manages, or directs the business.",
			},
			{
				Name:        "relationship-owner",
				In:          surface.InBody,
				Key:         "/relationship/owner",
				Kind:        surface.KindBoolean,
				Required:    true,
				Description: "Whether the representative owns any equity in the business.",
			},
			{
				Name:        "relationship-percent-ownership",
				In:          surface.InBody,
				Key:         "/relationship/percent_ownership",
				Kind:        surface.KindNumber,
				Description: "The representative's ownership percentage.",
				Format:      "double",
			},
			{
				Name:        "relationship-primary",
				In:          surface.InBody,
				Key:         "/relationship/primary",
				Kind:        surface.KindBoolean,
				Required:    true,
				Description: "Whether this person is the account's primary representative.",
			},
			{
				Name:        "relationship-title",
				In:          surface.InBody,
				Key:         "/relationship/title",
				Kind:        surface.KindString,
				Description: "The representative's job title.",
			},
			{
				Name:        "ssn-last4",
				In:          surface.InBody,
				Key:         "/ssn_last4",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Last four digits of the representative's Social Security number.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	})
}

func newRepresentativesUpdateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "representatives.update",
		OperationID: "updateRepresentative",
		Method:      "PUT",
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
				Name:        "dob",
				In:          surface.InBody,
				Key:         "/dob",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's date of birth in `YYYY-MM-DD` format.",
				Format:      "date",
			},
			{
				Name:        "email",
				In:          surface.InBody,
				Key:         "/email",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's email address.",
				Format:      "email",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Your unique ID for the representative.",
			},
			{
				Name:        "first-name",
				In:          surface.InBody,
				Key:         "/first_name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's first name.",
			},
			{
				Name:        "last-name",
				In:          surface.InBody,
				Key:         "/last_name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's last name.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs.",
			},
			{
				Name:        "mobile-number",
				In:          surface.InBody,
				Key:         "/mobile_number",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Representative's mobile phone number in E.164 format.",
			},
			{
				Name:        "relationship-control",
				In:          surface.InBody,
				Key:         "/relationship/control",
				Kind:        surface.KindBoolean,
				Required:    true,
				Description: "Whether the representative controls, manages, or directs the business.",
			},
			{
				Name:        "relationship-owner",
				In:          surface.InBody,
				Key:         "/relationship/owner",
				Kind:        surface.KindBoolean,
				Required:    true,
				Description: "Whether the representative owns any equity in the business.",
			},
			{
				Name:        "relationship-percent-ownership",
				In:          surface.InBody,
				Key:         "/relationship/percent_ownership",
				Kind:        surface.KindNumber,
				Description: "The representative's ownership percentage.",
				Format:      "double",
			},
			{
				Name:        "relationship-primary",
				In:          surface.InBody,
				Key:         "/relationship/primary",
				Kind:        surface.KindBoolean,
				Required:    true,
				Description: "Whether this person is the account's primary representative.",
			},
			{
				Name:        "relationship-title",
				In:          surface.InBody,
				Key:         "/relationship/title",
				Kind:        surface.KindString,
				Description: "The representative's job title.",
			},
			{
				Name:        "ssn-last4",
				In:          surface.InBody,
				Key:         "/ssn_last4",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Last four digits of the representative's Social Security number.",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: false,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "update <representative_id>",
		Short:   "Update a representative",
		Example: "  straddle representatives update <representative_id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "representatives.update",
			"straddle:operation-id": "updateRepresentative",
			"straddle:method":       "PUT",
			"straddle:path":         "/v1/representatives/{representative_id}",
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
	applyOverlay("representatives.update", cmd)
	return cmd
}
