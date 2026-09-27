// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint("payouts.upload-payout-authorization-proof", newPayoutsUploadPayoutAuthorizationProofCmd)
	registerSurface(surface.Surface{
		Endpoint:    "payouts.upload-payout-authorization-proof",
		OperationID: "uploadPayoutAuthorizationProof",
		Method:      "POST",
		Path:        "/v1/payouts/{id}/authorization",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "file",
				In:          surface.InForm,
				Key:         "File",
				Kind:        surface.KindFile,
				Required:    true,
				Enum:        []string{"application/pdf", "image/png", "image/jpeg", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
				Description: "The document file to upload as proof of authorization for this payout.",
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

func newPayoutsUploadPayoutAuthorizationProofCmd(flags *rootFlags) *cobra.Command {
	s := surface.Surface{
		Endpoint:    "payouts.upload-payout-authorization-proof",
		OperationID: "uploadPayoutAuthorizationProof",
		Method:      "POST",
		Path:        "/v1/payouts/{id}/authorization",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "file",
				In:          surface.InForm,
				Key:         "File",
				Kind:        surface.KindFile,
				Required:    true,
				Enum:        []string{"application/pdf", "image/png", "image/jpeg", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
				Description: "The document file to upload as proof of authorization for this payout.",
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
		Use:     "upload-payout-authorization-proof <id>",
		Short:   "Upload a proof-of-authorization document for a payout",
		Example: "  straddle payouts upload-payout-authorization-proof <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     "payouts.upload-payout-authorization-proof",
			"straddle:operation-id": "uploadPayoutAuthorizationProof",
			"straddle:method":       "POST",
			"straddle:path":         "/v1/payouts/{id}/authorization",
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
	applyOverlay("payouts.upload-payout-authorization-proof", cmd)
	return cmd
}
