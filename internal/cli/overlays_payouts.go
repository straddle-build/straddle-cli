// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("payouts.cancel", commandOverlay{
		short:   "Cancel a payout to prevent it from being processed. The status of the payout must be `created`, `scheduled`, or...",
		long:    "",
		example: "  straddle payouts cancel payout 550e8400-e29b-41d4-a716-446655440000 --reason example-value",
		aliases: []string{"payout", "update"},
		flags: []flagOverlay{
			{name: "reason", usage: "Details about why the payout status was updated."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "cancel",
	})
	registerCommandOverlay("payouts.create", commandOverlay{
		short:   "Use payouts to send money to your customers.",
		long:    "",
		example: "  straddle payouts create --currency example-value",
		flags: []flagOverlay{
			{name: "amount", usage: "The amount of the payout in cents."},
			{name: "config-auto-hold", usage: "Defines whether to automatically place this charge on hold after being created."},
			{name: "config-auto-hold-message", usage: "The reason the payout is being automatically held on creation."},
			{name: "currency", usage: "The currency of the payout. Only USD is supported."},
			{name: "description", usage: "An arbitrary description for the payout."},
			{name: "device-ip-address", usage: "The IP address of the device used when the customer authorized the charge or payout. Use `0.0.0.0` to represent an..."},
			{name: "external-id", usage: "Unique identifier for the payout in your database. This value must be unique across all payouts."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the payout in a..."},
			{name: "paykey", usage: "Value of the `paykey` used for the payout."},
			{name: "payment-date", usage: "The desired date on which the payout should be occur. For payouts, this means the date you want the funds to be sent..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "payouts",
	})
	registerCommandOverlay("payouts.get", commandOverlay{
		short:    "Retrieves the details of an existing payout. Supply the unique payout `id` to retrieve the corresponding payout...",
		long:     "",
		example:  "  straddle payouts get 550e8400-e29b-41d4-a716-446655440000",
		resource: "payouts",
	})
	registerCommandOverlay("payouts.get-unmasked-payout", commandOverlay{
		short:    "Get a payout by id.",
		long:     "",
		example:  "  straddle payouts unmask payouts-v1-get 550e8400-e29b-41d4-a716-446655440000",
		aliases:  []string{"payouts-v1-get", "get"},
		resource: "unmask",
	})
	registerCommandOverlay("payouts.hold", commandOverlay{
		short:   "Hold a payout to prevent it from being processed. The status of the payout must be `created`, `scheduled`, or `on_hold`.",
		long:    "",
		example: "  straddle payouts hold payout 550e8400-e29b-41d4-a716-446655440000 --reason example-value",
		aliases: []string{"payout", "update"},
		flags: []flagOverlay{
			{name: "reason", usage: "Details about why the payout status was updated."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "hold",
	})
	registerCommandOverlay("payouts.release", commandOverlay{
		short:   "Release a payout from a `hold` status to allow it to be rescheduled for processing.",
		long:    "",
		example: "  straddle payouts release payout 550e8400-e29b-41d4-a716-446655440000 --reason example-value",
		aliases: []string{"payout", "update"},
		flags: []flagOverlay{
			{name: "reason", usage: "Details about why the payout status was updated."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "release",
	})
	registerCommandOverlay("payouts.resubmit", commandOverlay{
		short:   "Resubmit a failed or reversed payout.",
		long:    "",
		example: "  straddle payouts resubmit create 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "description", usage: "Description."},
			{name: "external-id", usage: "External id."},
			{name: "payment-date", usage: "Payment date."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "resubmit",
	})
	registerCommandOverlay("payouts.update", commandOverlay{
		short:   "Update the details of a payout prior to processing. The status of the payout must be `created`, `scheduled`, or...",
		long:    "",
		example: "  straddle payouts update 550e8400-e29b-41d4-a716-446655440000 --description example-value",
		flags: []flagOverlay{
			{name: "amount", usage: "The amount of the payout in cents."},
			{name: "description", usage: "An arbitrary description for the payout."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the payout in a..."},
			{name: "payment-date", usage: "The desired date on which the payment should be occur. For payouts, this means the date you want the funds to be..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "payouts",
	})
}
