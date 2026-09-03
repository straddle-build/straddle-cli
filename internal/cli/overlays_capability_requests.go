// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("capability-requests.create", commandOverlay{
		short:   "Submits a request to enable a specific capability for an account. Use this endpoint to request additional features...",
		long:    "",
		example: "  straddle accounts capability-requests create 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "businesses-enable", usage: "Enable"},
			{name: "charges-daily-amount", usage: "The maximum dollar amount of charges in a calendar day."},
			{name: "charges-enable", usage: "Determines whether `charges` are enabled for the account."},
			{name: "charges-max-amount", usage: "The maximum amount of a single charge."},
			{name: "charges-monthly-amount", usage: "The maximum dollar amount of charges in a calendar month."},
			{name: "charges-monthly-count", usage: "The maximum number of charges in a calendar month."},
			{name: "individuals-enable", usage: "Enable"},
			{name: "internet-enable", usage: "Enable"},
			{name: "payouts-daily-amount", usage: "The maximum dollar amount of payouts in a day."},
			{name: "payouts-enable", usage: "Determines whether `payouts` are enabled for the account."},
			{name: "payouts-max-amount", usage: "The maximum amount of a single payout."},
			{name: "payouts-monthly-amount", usage: "The maximum dollar amount of payouts in a month."},
			{name: "payouts-monthly-count", usage: "The maximum number of payouts in a month."},
			{name: "signed-agreement-enable", usage: "Enable"},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "capability-requests",
	})
	registerCommandOverlay("capability-requests.list", commandOverlay{
		short:   "Retrieves a list of capability requests associated with an account. The requests are returned sorted by creation...",
		long:    "",
		example: "  straddle accounts capability-requests list 550e8400-e29b-41d4-a716-446655440000",
		aliases: []string{"list", "get"},
		flags: []flagOverlay{
			{name: "category", usage: "Filter capability requests by category. (one of: payment_type, customer_type, consent_type)", enumSet: true, enum: []string{"payment_type", "customer_type", "consent_type"}},
			{name: "page-number", usage: "Results page number. Starts at page 1."},
			{name: "page-size", usage: "Page size.Max value: 1000"},
			{name: "sort-by", usage: "Sort By."},
			{name: "sort-order", usage: "Sort Order. (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "status", usage: "Filter capability requests by their current status. (one of: active, inactive, in_review, rejected)", enumSet: true, enum: []string{"active", "inactive", "in_review", "rejected"}},
			{name: "type", usage: "Filter capability requests by the specific type of capability. (one of: charges, payouts, individuals, businesses, signed_agreement, internet)", enumSet: true, enum: []string{"charges", "payouts", "individuals", "businesses", "signed_agreement", "internet"}},
		},
		paginated: true,
		resource:  "capability-requests",
	})
}
