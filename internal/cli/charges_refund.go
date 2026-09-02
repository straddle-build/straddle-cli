// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	registerGeneratedEndpoint("charges.refund", newChargesRefundCmd)
}

func newChargesRefundCmd(flags *rootFlags) *cobra.Command {
	var flagRequestIdHeader string
	var flagCorrelationIdHeader string
	var flagIdempotencyKeyHeader string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "refund <id>",
		Short:       "Refund a paid charge",
		Example:     "  straddle charges refund <id>",
		Annotations: map[string]string{"straddle:endpoint": "charges.refund", "straddle:operation-id": "refundCharge", "straddle:method": "POST", "straddle:path": "/v1/charges/{id}/refund"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			path := "/v1/charges/{id}/refund"
			path = replacePathParam(path, "id", args[0])
			params := map[string]string{}
			headers := map[string]string{}
			if cmd.Flags().Changed("request-id") {
				headers["Request-Id"] = flagRequestIdHeader
			}
			if cmd.Flags().Changed("correlation-id") {
				headers["Correlation-Id"] = flagCorrelationIdHeader
			}
			if cmd.Flags().Changed("idempotency-key") {
				headers["Idempotency-Key"] = flagIdempotencyKeyHeader
			}
			var body map[string]any
			if stdinBody {
				stdinData, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				if err := json.Unmarshal(stdinData, &body); err != nil {
					return fmt.Errorf("parsing stdin JSON: %w", err)
				}
			} else {
				body = map[string]any{}
			}
			data, statusCode, err := c.PostWithParamsAndHeaders(path, params, body, headers)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printGeneratedMutationOutput(cmd, flags, "POST", "charges.refund", path, statusCode, data)
		},
	}
	cmd.Flags().StringVar(&flagRequestIdHeader, "request-id", "", "Optional client-generated identifier for tracing one request.")
	cmd.Flags().StringVar(&flagCorrelationIdHeader, "correlation-id", "", "Optional client-generated identifier for tracing a series of related requests.")
	cmd.Flags().StringVar(&flagIdempotencyKeyHeader, "idempotency-key", "", "Optional client-generated key for an idempotent request.")
	cmd.Flags().BoolVar(&stdinBody, "stdin", false, "Read JSON request body from stdin")
	return cmd
}
