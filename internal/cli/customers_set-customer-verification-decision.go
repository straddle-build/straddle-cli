// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("customers.set-customer-verification-decision", newCustomersSetCustomerVerificationDecisionCmd)
	registerSurface(surface.Surface{
		Endpoint:    "customers.set-customer-verification-decision",
		OperationID: "setCustomerVerificationDecision",
		Method:      "PATCH",
		Path:        "/v1/customers/{id}/review",
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
				Name:        "status",
				In:          surface.InBody,
				Key:         "/status",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"verified", "rejected"},
				Description: "The final status of the customer review.",
				Default:     "verified",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newCustomersSetCustomerVerificationDecisionCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "customers.set-customer-verification-decision",
		OperationID: "setCustomerVerificationDecision",
		Method:      "PATCH",
		Path:        "/v1/customers/{id}/review",
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
				Name:        "status",
				In:          surface.InBody,
				Key:         "/status",
				Kind:        surface.KindString,
				Required:    true,
				Enum:        []string{"verified", "rejected"},
				Description: "The final status of the customer review.",
				Default:     "verified",
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "set-customer-verification-decision <id>",
		Short:   "Set a customer verification decision",
		Example: "  straddle customers set-customer-verification-decision <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "customers.set-customer-verification-decision",
			"straddle:operation-id": "setCustomerVerificationDecision",
			"straddle:method":       "PATCH",
			"straddle:path":         "/v1/customers/{id}/review",
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
	applyOverlay("customers.set-customer-verification-decision", cmd)
	return cmd
}
