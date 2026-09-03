// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("account-settings.get", commandOverlay{
		short:    "Shortcut for 'account-settings get-settings'. Get all resolved settings for the specified account, including inherited values from organization, platform, and...",
		long:     "",
		example:  "  straddle account-settings 550e8400-e29b-41d4-a716-446655440000",
		resource: "account-settings",
		unwrap:   true,
	})
	registerCommandOverlay("accounts.create", commandOverlay{
		short:   "Creates a new account associated with your Straddle platform integration. This endpoint allows you to set up an...",
		long:    "",
		example: "  straddle accounts create --access-level standard",
		flags: []flagOverlay{
			{name: "access-level", usage: "The access level granted to the account. This is determined by your platform configuration. Use `standard` unless..."},
			{name: "account-type", usage: "The type of account to be created. Currently, only `business` is supported."},
			{name: "business-profile-address-country", usage: "The country of the address, in ISO 3166-1 alpha-2 format."},
			{name: "business-profile-address-line1", usage: "Primary address line (e.g., street, PO Box)."},
			{name: "business-profile-address-line2", usage: "Secondary address line (e.g., apartment, suite, unit, or building)."},
			{name: "business-profile-description", usage: "A brief description of the business and its products or services."},
			{name: "business-profile-industry-category", usage: "The general category of the industry. Required if not providing MCC."},
			{name: "business-profile-industry-mcc", usage: "The Merchant Category Code (MCC) that best describes the business. Optional."},
			{name: "business-profile-industry-sector", usage: "The specific sector within the industry category. Required if not providing MCC."},
			{name: "business-profile-phone", usage: "The primary contact phone number for the business."},
			{name: "business-profile-support-channels-email", usage: "The email address for customer support inquiries."},
			{name: "business-profile-support-channels-phone", usage: "The phone number for customer support."},
			{name: "business-profile-support-channels-url", usage: "The URL of the business's customer support page or contact form."},
			{name: "business-profile-tax-id", usage: "The business's tax identification number (e.g., EIN in the US)."},
			{name: "business-profile-use-case", usage: "A description of how the business intends to use Straddle's services."},
			{name: "business-profile-website", usage: "URL of the business's primary marketing website."},
			{name: "external-id", usage: "Unique identifier for the account in your database, used for cross-referencing between Straddle and your systems."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the account in a..."},
			{name: "organization-id", usage: "The unique identifier of the organization related to this account."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "accounts",
	})
	registerCommandOverlay("accounts.get", commandOverlay{
		short:    "Retrieves the details of an account that has previously been created. Supply the unique account ID that was returned...",
		long:     "",
		example:  "  straddle accounts get 550e8400-e29b-41d4-a716-446655440000",
		resource: "accounts",
	})
	registerCommandOverlay("accounts.list", commandOverlay{
		short:   "Returns a list of accounts associated with your Straddle platform integration. The accounts are returned sorted by...",
		long:    "",
		example: "  straddle accounts list",
		flags: []flagOverlay{
			{name: "external-id", usage: "Filter accounts by their external ID."},
			{name: "page-number", usage: "Results page number. Starts at page 1. Default value: 1"},
			{name: "page-size", usage: "Page size. Default value: 100. Max value: 1000"},
			{name: "search-text", usage: "Search text"},
			{name: "sort-by", usage: "Sort By. Default value: 'id'."},
			{name: "sort-order", usage: "Sort Order. Default value: 'asc'. (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
			{name: "status", usage: "Status (one of: created, onboarding, active, rejected, inactive)", enumSet: true, enum: []string{"created", "onboarding", "active", "rejected", "inactive"}},
			{name: "type", usage: "Type (one of: business)", enumSet: true, enum: []string{"business"}},
		},
		paginated: true,
		resource:  "accounts",
	})
	registerCommandOverlay("accounts.onboard", commandOverlay{
		short:   "Initiates the onboarding process for a new account. This endpoint can only be used for accounts where at least one...",
		long:    "",
		example: "  straddle accounts onboard account 550e8400-e29b-41d4-a716-446655440000",
		aliases: []string{"account", "create"},
		flags: []flagOverlay{
			{name: "stdin", usage: "Read request body as JSON from stdin"},
			{name: "terms-of-service-accepted-date", usage: "The datetime of when the terms of service were accepted, in ISO 8601 format."},
			{name: "terms-of-service-accepted-ip", usage: "The IP address from which the terms of service were accepted."},
			{name: "terms-of-service-accepted-user-agent", usage: "The user agent string of the browser or application used to accept the terms."},
			{name: "terms-of-service-agreement-type", usage: "The type or version of the agreement accepted. Use `embedded` unless your platform was specifically enabled for...", defaultSet: true, defaultVal: ""},
			{name: "terms-of-service-agreement-url", usage: "The URL where the full text of the accepted agreement can be found."},
		},
		resource: "onboard",
	})
	registerCommandOverlay("accounts.simulate-account-onboarding", commandOverlay{
		short:   "Simulate the status transitions for sandbox accounts. This endpoint can only be used for sandbox accounts.",
		long:    "",
		example: "  straddle accounts simulate create 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "final-status", usage: "Final status (one of: onboarding, active)", enumSet: true, enum: []string{"onboarding", "active"}},
		},
		body:     true,
		resource: "simulate",
	})
	registerCommandOverlay("accounts.update", commandOverlay{
		short:   "Updates an existing account's information. This endpoint allows you to update various account details during...",
		long:    "",
		example: "  straddle accounts update 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "business-profile-address-country", usage: "The country of the address, in ISO 3166-1 alpha-2 format."},
			{name: "business-profile-address-line1", usage: "Primary address line (e.g., street, PO Box)."},
			{name: "business-profile-address-line2", usage: "Secondary address line (e.g., apartment, suite, unit, or building)."},
			{name: "business-profile-description", usage: "A brief description of the business and its products or services."},
			{name: "business-profile-industry-category", usage: "The general category of the industry. Required if not providing MCC."},
			{name: "business-profile-industry-mcc", usage: "The Merchant Category Code (MCC) that best describes the business. Optional."},
			{name: "business-profile-industry-sector", usage: "The specific sector within the industry category. Required if not providing MCC."},
			{name: "business-profile-phone", usage: "The primary contact phone number for the business."},
			{name: "business-profile-support-channels-email", usage: "The email address for customer support inquiries."},
			{name: "business-profile-support-channels-phone", usage: "The phone number for customer support."},
			{name: "business-profile-support-channels-url", usage: "The URL of the business's customer support page or contact form."},
			{name: "business-profile-tax-id", usage: "The business's tax identification number (e.g., EIN in the US)."},
			{name: "business-profile-use-case", usage: "A description of how the business intends to use Straddle's services."},
			{name: "business-profile-website", usage: "URL of the business's primary marketing website."},
			{name: "external-id", usage: "Unique identifier for the account in your database, used for cross-referencing between Straddle and your systems."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the account in a..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "accounts",
	})
}
