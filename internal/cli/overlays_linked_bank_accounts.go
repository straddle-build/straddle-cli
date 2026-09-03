// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("linked-bank-accounts.cancel", commandOverlay{
		short:    "Cancels an existing linked bank account. This can be used to cancel a linked bank account before it has been...",
		long:     "",
		example:  "  straddle linked-bank-accounts cancel update 550e8400-e29b-41d4-a716-446655440000",
		body:     true,
		resource: "cancel",
	})
	registerCommandOverlay("linked-bank-accounts.create", commandOverlay{
		short:   "Creates a new linked bank account associated with a Straddle account. This endpoint allows you to associate external...",
		long:    "",
		example: "  straddle linked-bank-accounts create --account-id 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "account-id", usage: "The unique identifier of the Straddle account to associate this bank account with."},
			{name: "bank-account-account-holder", usage: "The name of the account holder as it appears on the bank account. Typically, this is the legal name of the business..."},
			{name: "bank-account-routing-number", usage: "The routing number of the bank account."},
			{name: "description", usage: "Optional description for the bank account."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the linked bank..."},
			{name: "platform-id", usage: "The unique identifier of the Straddle Platform to associate this bank account with."},
			{name: "purposes", usage: "The purposes for the linked bank account."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "linked-bank-accounts",
	})
	registerCommandOverlay("linked-bank-accounts.get", commandOverlay{
		short:    "Retrieves the details of a linked bank account that has previously been created. Supply the unique linked bank...",
		long:     "",
		example:  "  straddle linked-bank-accounts get 550e8400-e29b-41d4-a716-446655440000",
		resource: "linked-bank-accounts",
	})
	registerCommandOverlay("linked-bank-accounts.get-unmasked-linked-bank-account", commandOverlay{
		short:    "Retrieves the unmasked details of a linked bank account that has previously been created. Supply the unique linked...",
		long:     "",
		example:  "  straddle linked-bank-accounts unmask get-linked-bank-account-unmasked 550e8400-e29b-41d4-a716-446655440000",
		aliases:  []string{"get-linked-bank-account-unmasked", "get"},
		resource: "unmask",
	})
	registerCommandOverlay("linked-bank-accounts.list", commandOverlay{
		short:   "Returns a list of bank accounts associated with a specific Straddle account. The linked bank accounts are returned...",
		long:    "",
		example: "  straddle linked-bank-accounts list",
		flags: []flagOverlay{
			{name: "account-id", usage: "The unique identifier of the related account."},
			{name: "level", usage: "Level (one of: account, platform)", enumSet: true, enum: []string{"account", "platform"}},
			{name: "page-number", usage: "Results page number. Starts at page 1."},
			{name: "page-size", usage: "Page size. Max value: 1000"},
			{name: "purpose", usage: "The purpose of the linked bank accounts to return. Possible values: 'charges', 'payouts', 'billing'. (one of: charges, payouts, billing)", enumSet: true, enum: []string{"charges", "payouts", "billing"}},
			{name: "sort-by", usage: "Sort By."},
			{name: "sort-order", usage: "Sort Order. (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "status", usage: "The status of the linked bank accounts to return. Possible values: 'created', 'onboarding', 'active', 'inactive',... (one of: created, onboarding, active, rejected, inactive, canceled)", enumSet: true, enum: []string{"created", "onboarding", "active", "rejected", "inactive", "canceled"}},
		},
		paginated: true,
		resource:  "linked-bank-accounts",
	})
	registerCommandOverlay("linked-bank-accounts.update", commandOverlay{
		short:   "Updates an existing linked bank account's information. This can be used to update account details during onboarding...",
		long:    "",
		example: "  straddle linked-bank-accounts update 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "bank-account-account-holder", usage: "The name of the account holder as it appears on the bank account. Typically, this is the legal name of the business..."},
			{name: "bank-account-routing-number", usage: "The routing number of the bank account."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the linked bank..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "linked-bank-accounts",
	})
}
