// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import "github.com/spf13/cobra"

type requestHeaderFlags struct {
	correlationID  string
	idempotencyKey string
	requestID      string
}

func (h *requestHeaderFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&h.correlationID, "correlation-id", "", "Optional client-generated identifier for tracing a series of related requests.")
	cmd.Flags().StringVar(&h.idempotencyKey, "idempotency-key", "", "Optional client-generated key for an idempotent request.")
	cmd.Flags().StringVar(&h.requestID, "request-id", "", "Optional client-generated identifier for tracing one request.")
}

func (h *requestHeaderFlags) headers() map[string]string {
	headers := map[string]string{}
	if h.correlationID != "" {
		headers["Correlation-Id"] = h.correlationID
	}
	if h.idempotencyKey != "" {
		headers["Idempotency-Key"] = h.idempotencyKey
	}
	if h.requestID != "" {
		headers["Request-Id"] = h.requestID
	}
	return headers
}
