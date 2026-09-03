// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("bridge.create-bank-account-paykey", commandOverlay{
		short:   "Use Bridge to create a new paykey using a bank routing and account number as the source. This endpoint allows you to...",
		long:    "",
		example: "  straddle bridge create-bank-account-paykey --account-number example-value",
		flags: []flagOverlay{
			{name: "account-number", usage: "The bank account number."},
			{name: "account-type", usage: "Account type"},
			{name: "config-processing-method", usage: "Processing method"},
			{name: "config-sandbox-outcome", usage: "Sandbox outcome"},
			{name: "customer-id", usage: "Unique identifier of the related customer object."},
			{name: "external-id", usage: "Unique identifier for the paykey in your database, used for cross-referencing between Straddle and your systems."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the paykey in a..."},
			{name: "routing-number", usage: "The routing number of the bank account."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "bridge",
	})
	registerCommandOverlay("bridge.create-bridge-token", commandOverlay{
		short:   "Use this endpoint to generate a session token for use in the Bridge widget.",
		long:    "",
		example: "  straddle bridge create-token --customer-id 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "config-processing-method", usage: "Processing method"},
			{name: "config-sandbox-outcome", usage: "Sandbox outcome"},
			{name: "customer-id", usage: "The Straddle generated unique identifier of the `customer` to create a bridge token for."},
			{name: "external-id", usage: "Unique identifier for the paykey in your database, used for cross-referencing between Straddle and your systems."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "bridge",
	})
	registerCommandOverlay("bridge.create-plaid-paykey", commandOverlay{
		short:   "Use Bridge to create a new paykey using a Plaid token as the source. This endpoint allows you to create a secure...",
		long:    "",
		example: "  straddle bridge create-plaid-paykey --customer-id 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "config-processing-method", usage: "Processing method"},
			{name: "config-sandbox-outcome", usage: "Sandbox outcome"},
			{name: "customer-id", usage: "Unique identifier of the related customer object."},
			{name: "external-id", usage: "Unique identifier for the paykey in your database, used for cross-referencing between Straddle and your systems."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the paykey in a..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "bridge",
	})
	registerCommandOverlay("bridge.create-quiltt-paykey", commandOverlay{
		short:   "Creates a new paykey using a Quiltt token as the source. This endpoint allows you to create a secure payment token...",
		long:    "",
		example: "  straddle bridge create --customer-id 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "config-processing-method", usage: "Processing method"},
			{name: "config-sandbox-outcome", usage: "Sandbox outcome"},
			{name: "customer-id", usage: "Unique identifier of the related customer object."},
			{name: "external-id", usage: "Unique identifier for the paykey in your database, used for cross-referencing between Straddle and your systems."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the paykey in a..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "bridge",
	})
}
