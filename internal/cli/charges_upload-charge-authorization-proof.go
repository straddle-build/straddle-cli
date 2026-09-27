// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("charges.upload-charge-authorization-proof", newChargesUploadChargeAuthorizationProofCmd)
	registerSurface(surface.Surface{
		Endpoint:    "charges.upload-charge-authorization-proof",
		OperationID: "uploadChargeAuthorizationProof",
		Method:      "POST",
		Path:        "/v1/charges/{id}/authorization",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "file",
				In:          surface.InForm,
				Key:         "File",
				Kind:        surface.KindFile,
				Required:    true,
				Enum:        []string{"application/pdf", "image/png", "image/jpeg", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
				Description: "The document file to upload as proof of authorization for this charge.",
			},
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
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	})
}

func newChargesUploadChargeAuthorizationProofCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "charges.upload-charge-authorization-proof",
		OperationID: "uploadChargeAuthorizationProof",
		Method:      "POST",
		Path:        "/v1/charges/{id}/authorization",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "file",
				In:          surface.InForm,
				Key:         "File",
				Kind:        surface.KindFile,
				Required:    true,
				Enum:        []string{"application/pdf", "image/png", "image/jpeg", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
				Description: "The document file to upload as proof of authorization for this charge.",
			},
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
		},
		HasBody:              true,
		BodyRequired:         true,
		AcceptsAccountHeader: true,
		ReadOnly:             false,
	}
	cmd := &cobra.Command{
		Use:     "upload-charge-authorization-proof <id>",
		Short:   "Upload a proof-of-authorization document for a charge",
		Example: "  straddle charges upload-charge-authorization-proof <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "charges.upload-charge-authorization-proof",
			"straddle:operation-id": "uploadChargeAuthorizationProof",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/charges/{id}/authorization",
		},
	}
	bind := bindSurface(cmd, s)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		req, err := bind(args)
		if err != nil {
			return err
		}
		return executeSurface(cmd, flags, s, req)
	}
	applyOverlay("charges.upload-charge-authorization-proof", cmd)
	return cmd
}
