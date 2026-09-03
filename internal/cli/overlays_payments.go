// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("payments.list", commandOverlay{
		short:   "Shortcut for 'payments list'. Search for payments, including `charges` and `payouts`, using a variety of criteria. This endpoint supports advanced...",
		long:    "",
		example: "  straddle payments",
		flags: []flagOverlay{
			{name: "customer-id", usage: "Search using the `customer_id` of a `charge` or `payout`."},
			{name: "default-page-size", usage: "Default page size"},
			{name: "default-sort", usage: "Default sort (one of: created_at, payment_date, effective_at, id, amount)", enumSet: true, enum: []string{"created_at", "payment_date", "effective_at", "id", "amount"}},
			{name: "default-sort-order", usage: "Default sort order (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "external-id", usage: "Search using the `external_id` of a `charge` or `payout`."},
			{name: "funding-id", usage: "Search using the `funding_id` of a `charge` or `payout`."},
			{name: "include-metadata", usage: "Include the metadata for payments in the returned data."},
			{name: "max-amount", usage: "Search using a maximum `amount` of a `charge` or `payout`."},
			{name: "max-created-at", usage: "Search using the latest `created_at` date of a `charge` or `payout`."},
			{name: "max-effective-at", usage: "Search using the latest `effective_date` of a `charge` or `payout`."},
			{name: "max-payment-date", usage: "Search using the latest `payment_date` of a `charge` or `payout`."},
			{name: "min-amount", usage: "Search using the minimum `amount of a `charge` or `payout`."},
			{name: "min-created-at", usage: "Search using the earliest `created_at` date of a `charge` or `payout`."},
			{name: "min-effective-at", usage: "Search using the earliest `effective_date` of a `charge` or `payout`."},
			{name: "min-payment-date", usage: "Search using the earliest ` `of a `charge` or `payout`."},
			{name: "page-number", usage: "Results page number. Starts at page 1."},
			{name: "page-size", usage: "Results page size. Max value: 1000"},
			{name: "paykey", usage: "Search using the `paykey` of a `charge` or `payout`."},
			{name: "paykey-id", usage: "Search using the `paykey_id` of a `charge` or `payout`."},
			{name: "payment-id", usage: "Search using the `id` of a `charge` or `payout`."},
			{name: "payment-status", usage: "Search by the status of a `charge` or `payout`.", defaultSet: true, defaultVal: ""},
			{name: "payment-type", usage: "Search by the type of a `charge` or `payout`.", defaultSet: true, defaultVal: ""},
			{name: "search-text", usage: "Search using a text string associated with a `charge` or `payout`."},
			{name: "sort-by", usage: "Sort by (one of: created_at, payment_date, effective_at, id, amount)", enumSet: true, enum: []string{"created_at", "payment_date", "effective_at", "id", "amount"}},
			{name: "sort-order", usage: "Sort order (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "status-reason", usage: "Reason for latest payment status change.", defaultSet: true, defaultVal: ""},
			{name: "status-source", usage: "Source of latest payment status change.", defaultSet: true, defaultVal: ""},
		},
		paginated: true,
		resource:  "payments",
		unwrap:    true,
	})
}
