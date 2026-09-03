// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("funding-events.get", commandOverlay{
		short:    "Retrieves the details of an existing funding event. Supply the unique funding event `id`, and Straddle will return...",
		long:     "",
		example:  "  straddle funding-events get 550e8400-e29b-41d4-a716-446655440000",
		resource: "funding-events",
	})
	registerCommandOverlay("funding-events.list", commandOverlay{
		short:   "Retrieves a list of funding events for your account. This endpoint supports advanced sorting and filtering options.",
		long:    "",
		example: "  straddle funding-events list",
		flags: []flagOverlay{
			{name: "created-from", usage: "The start date of the range to filter by using the `YYYY-MM-DD` format."},
			{name: "created-to", usage: "The end date of the range to filter by using the `YYYY-MM-DD` format."},
			{name: "direction", usage: "Direction (one of: deposit, withdrawal)", enumSet: true, enum: []string{"deposit", "withdrawal"}},
			{name: "event-type", usage: "Event type (one of: charge_deposit, charge_reversal, payout_return, payout_withdrawal)", enumSet: true, enum: []string{"charge_deposit", "charge_reversal", "payout_return", "payout_withdrawal"}},
			{name: "page-number", usage: "Results page number. Starts at page 1."},
			{name: "page-size", usage: "Results page size. Max value: 1000"},
			{name: "search-text", usage: "Search text."},
			{name: "sort-by", usage: "Sort by (one of: transfer_date, id, amount)", enumSet: true, enum: []string{"transfer_date", "id", "amount"}},
			{name: "sort-order", usage: "Sort order (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "status", usage: "Funding Event status.", defaultSet: true, defaultVal: ""},
			{name: "status-reason", usage: "Reason for latest payment status change.", defaultSet: true, defaultVal: ""},
			{name: "status-source", usage: "Source of latest payment status change.", defaultSet: true, defaultVal: ""},
			{name: "trace-id", usage: "Trace Id."},
			{name: "trace-number", usage: "Trace number."},
		},
		paginated: true,
		resource:  "funding-events",
	})
	registerCommandOverlay("funding-events.list-funding-event-payments", commandOverlay{
		short:   "Shortcut for 'funding-event-payments get'. All the payments that made up the funding event",
		long:    "",
		example: "  straddle funding-event-payments 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "default-page-size", usage: "Default page size"},
			{name: "default-sort", usage: "Default sort (one of: created_at, payment_date, effective_at, id)", enumSet: true, enum: []string{"created_at", "payment_date", "effective_at", "id"}},
			{name: "default-sort-order", usage: "Default sort order (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "include-metadata", usage: "Include the metadata for payments in the returned data."},
			{name: "page-number", usage: "Results page number. Starts at page 1. Default value: 1"},
			{name: "page-size", usage: "Results page size. Default value: 100. Max value: 1000"},
			{name: "sort-by", usage: "Sort by (one of: created_at, payment_date, effective_at, id)", enumSet: true, enum: []string{"created_at", "payment_date", "effective_at", "id"}},
			{name: "sort-order", usage: "Sort order (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
		},
		paginated: true,
		resource:  "funding-event-payments",
		unwrap:    true,
	})
	registerCommandOverlay("funding-events.simulate", commandOverlay{
		short:   "Simulate a funding event for testing. This endpoint can only be used in the sandbox environment.",
		long:    "",
		example: "  straddle funding-events create --funding-event-job-type charges",
		flags: []flagOverlay{
			{name: "funding-event-job-type", usage: "Supported job types are Charges and Payouts"},
			{name: "sandbox-outcome", usage: "Payment will simulate processing if not Standard."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "funding-events",
	})
}
