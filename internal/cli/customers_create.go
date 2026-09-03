// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("customers.create", newCustomersCreateCmd)
	registerSurface(surface.Surface{
		Endpoint:    "customers.create",
		OperationID: "createCustomer",
		Method:      "POST",
		Path:        "/v1/customers",
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
				Name:        "address-address1",
				In:          surface.InBody,
				Key:         "/address/address1",
				Kind:        surface.KindString,
				Description: "Primary address line, such as a street address or PO Box.",
			},
			{
				Name:        "address-address2",
				In:          surface.InBody,
				Key:         "/address/address2",
				Kind:        surface.KindString,
				Description: "Secondary address line, such as an apartment, suite, unit, or building.",
			},
			{
				Name:        "address-city",
				In:          surface.InBody,
				Key:         "/address/city",
				Kind:        surface.KindString,
				Description: "City, district, suburb, town, or village.",
			},
			{
				Name:        "address-state",
				In:          surface.InBody,
				Key:         "/address/state",
				Kind:        surface.KindString,
				Description: "Two-letter state code.",
			},
			{
				Name:        "address-zip",
				In:          surface.InBody,
				Key:         "/address/zip",
				Kind:        surface.KindString,
				Description: "ZIP or postal code.",
			},
			{
				Name:        "compliance-profile",
				In:          surface.InBody,
				Key:         "/compliance_profile",
				Kind:        surface.KindJSON,
				Description: "Customer compliance profile.",
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
				Enum: []string{"standard", "verified", "rejected", "review"},
			},
			{
				Name:        "device-ip-address",
				In:          surface.InBody,
				Key:         "/device/ip_address",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Customer IP address at profile creation.",
				Format:      "ipv4",
			},
			{
				Name:        "email",
				In:          surface.InBody,
				Key:         "/email",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Customer email address.",
				Format:      "email",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Unique identifier for the customer in your system.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs associated with the customer.",
			},
			{
				Name:        "name",
				In:          surface.InBody,
				Key:         "/name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Full name for an individual customer or business name for a business customer.",
			},
			{
				Name:        "phone",
				In:          surface.InBody,
				Key:         "/phone",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Customer phone number in E.164 format.",
			},
			{
				Name:     "type",
				In:       surface.InBody,
				Key:      "/type",
				Kind:     surface.KindString,
				Required: true,
				Enum:     []string{"individual", "business"},
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newCustomersCreateCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "customers.create",
		OperationID: "createCustomer",
		Method:      "POST",
		Path:        "/v1/customers",
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
				Name:        "address-address1",
				In:          surface.InBody,
				Key:         "/address/address1",
				Kind:        surface.KindString,
				Description: "Primary address line, such as a street address or PO Box.",
			},
			{
				Name:        "address-address2",
				In:          surface.InBody,
				Key:         "/address/address2",
				Kind:        surface.KindString,
				Description: "Secondary address line, such as an apartment, suite, unit, or building.",
			},
			{
				Name:        "address-city",
				In:          surface.InBody,
				Key:         "/address/city",
				Kind:        surface.KindString,
				Description: "City, district, suburb, town, or village.",
			},
			{
				Name:        "address-state",
				In:          surface.InBody,
				Key:         "/address/state",
				Kind:        surface.KindString,
				Description: "Two-letter state code.",
			},
			{
				Name:        "address-zip",
				In:          surface.InBody,
				Key:         "/address/zip",
				Kind:        surface.KindString,
				Description: "ZIP or postal code.",
			},
			{
				Name:        "compliance-profile",
				In:          surface.InBody,
				Key:         "/compliance_profile",
				Kind:        surface.KindJSON,
				Description: "Customer compliance profile.",
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
				Enum: []string{"standard", "verified", "rejected", "review"},
			},
			{
				Name:        "device-ip-address",
				In:          surface.InBody,
				Key:         "/device/ip_address",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Customer IP address at profile creation.",
				Format:      "ipv4",
			},
			{
				Name:        "email",
				In:          surface.InBody,
				Key:         "/email",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Customer email address.",
				Format:      "email",
			},
			{
				Name:        "external-id",
				In:          surface.InBody,
				Key:         "/external_id",
				Kind:        surface.KindString,
				Description: "Unique identifier for the customer in your system.",
			},
			{
				Name:        "metadata",
				In:          surface.InBody,
				Key:         "/metadata",
				Kind:        surface.KindJSON,
				Description: "Up to 20 user-defined key-value pairs associated with the customer.",
			},
			{
				Name:        "name",
				In:          surface.InBody,
				Key:         "/name",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Full name for an individual customer or business name for a business customer.",
			},
			{
				Name:        "phone",
				In:          surface.InBody,
				Key:         "/phone",
				Kind:        surface.KindString,
				Required:    true,
				Description: "Customer phone number in E.164 format.",
			},
			{
				Name:     "type",
				In:       surface.InBody,
				Key:      "/type",
				Kind:     surface.KindString,
				Required: true,
				Enum:     []string{"individual", "business"},
			},
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a customer",
		Example: "  straddle customers create",
		Annotations: map[string]string{
			"straddle:endpoint":     "customers.create",
			"straddle:operation-id": "createCustomer",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/customers",
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
	applyOverlay("customers.create", cmd)
	return cmd
}
