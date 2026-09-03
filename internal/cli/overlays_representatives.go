// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("representatives.create", commandOverlay{
		short:   "Creates a new representative associated with an account. Representatives are individuals who have legal authority or...",
		long:    "",
		example: "  straddle representatives create --account-id 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "account-id", usage: "The unique identifier of the account this representative is associated with."},
			{name: "dob", usage: "Date of birth for the representative in ISO 8601 format (YYYY-MM-DD)."},
			{name: "email", usage: "The company email address of the representative."},
			{name: "external-id", usage: "Unique identifier for the representative in your database, used for cross-referencing between Straddle and your systems."},
			{name: "first-name", usage: "The first name of the representative."},
			{name: "last-name", usage: "The last name of the representative."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the represetative..."},
			{name: "mobile-number", usage: "The mobile phone number of the representative."},
			{name: "relationship-control", usage: "Whether the representative has significant responsibility to control, manage, or direct the organization. One..."},
			{name: "relationship-owner", usage: "Whether the representative owns any percentage of of the equity interests of the legal entity."},
			{name: "relationship-percent-ownership", usage: "The percentage of ownership the representative has. Required if 'Owner' is true."},
			{name: "relationship-primary", usage: "Whether the person is authorized as the primary representative of the account. This is the person chosen by the..."},
			{name: "relationship-title", usage: "The job title of the representative."},
			{name: "ssn-last4", usage: "The last 4 digits of the representative's Social Security Number."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "representatives",
	})
	registerCommandOverlay("representatives.get", commandOverlay{
		short:    "Retrieves the details of an existing representative. Supply the unique representative ID, and Straddle will return...",
		long:     "",
		example:  "  straddle representatives get 550e8400-e29b-41d4-a716-446655440000",
		resource: "representatives",
	})
	registerCommandOverlay("representatives.get-unmasked-representative", commandOverlay{
		short:    "Retrieves the unmasked details of a representative that has previously been created. Supply the unique...",
		long:     "",
		example:  "  straddle representatives unmask get 550e8400-e29b-41d4-a716-446655440000",
		resource: "unmask",
	})
	registerCommandOverlay("representatives.list", commandOverlay{
		short:   "Returns a list of representatives associated with a specific account or organization. The representatives are...",
		long:    "",
		example: "  straddle representatives list",
		flags: []flagOverlay{
			{name: "account-id", usage: "The unique identifier of the account to list representatives for."},
			{name: "level", usage: "Level (one of: account, platform)", enumSet: true, enum: []string{"account", "platform"}},
			{name: "organization-id", usage: "Organization id"},
			{name: "page-number", usage: "Results page number. Starts at page 1."},
			{name: "page-size", usage: "Page size. Max value: 1000"},
			{name: "platform-id", usage: "Platform id"},
			{name: "sort-by", usage: "Sort By."},
			{name: "sort-order", usage: "Sort Order. (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
		},
		paginated: true,
		resource:  "representatives",
	})
	registerCommandOverlay("representatives.update", commandOverlay{
		short:   "Updates an existing representative's information. This can be used to update personal details, contact information,...",
		long:    "",
		example: "  straddle representatives update 550e8400-e29b-41d4-a716-446655440000 --dob 2026-01-15",
		flags: []flagOverlay{
			{name: "dob", usage: "The date of birth of the representative, in ISO 8601 format (YYYY-MM-DD)."},
			{name: "email", usage: "The email address of the representative."},
			{name: "external-id", usage: "Unique identifier for the representative in your database, used for cross-referencing between Straddle and your systems."},
			{name: "first-name", usage: "The first name of the representative."},
			{name: "last-name", usage: "The last name of the representative."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the represetative..."},
			{name: "mobile-number", usage: "The mobile phone number of the representative."},
			{name: "relationship-control", usage: "Whether the representative has significant responsibility to control, manage, or direct the organization. One..."},
			{name: "relationship-owner", usage: "Whether the representative owns any percentage of of the equity interests of the legal entity."},
			{name: "relationship-percent-ownership", usage: "The percentage of ownership the representative has. Required if 'Owner' is true."},
			{name: "relationship-primary", usage: "Whether the person is authorized as the primary representative of the account. This is the person chosen by the..."},
			{name: "relationship-title", usage: "The job title of the representative."},
			{name: "ssn-last4", usage: "The last 4 digits of the representative's Social Security Number."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "representatives",
	})
}
