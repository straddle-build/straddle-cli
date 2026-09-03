// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("organizations.create", commandOverlay{
		short:   "Creates a new organization related to your Straddle integration. Organizations can be used to group related accounts...",
		long:    "",
		example: "  straddle organizations create --name example-resource",
		flags: []flagOverlay{
			{name: "external-id", usage: "Unique identifier for the organization in your database, used for cross-referencing between Straddle and your systems."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the organization..."},
			{name: "name", usage: "The name of the organization."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "organizations",
	})
	registerCommandOverlay("organizations.get", commandOverlay{
		short:    "Retrieves the details of an Organization that has previously been created. Supply the unique organization ID that...",
		long:     "",
		example:  "  straddle organizations get-by-id 550e8400-e29b-41d4-a716-446655440000",
		aliases:  []string{"get-by-id", "get"},
		resource: "organizations",
	})
	registerCommandOverlay("organizations.list", commandOverlay{
		short:   "Retrieves a list of organizations associated with your Straddle integration. The organizations are returned sorted...",
		long:    "",
		example: "  straddle organizations list",
		flags: []flagOverlay{
			{name: "external-id", usage: "List organizations by their external ID."},
			{name: "name", usage: "List organizations by name (partial match supported)."},
			{name: "page-number", usage: "Results page number. Starts at page 1."},
			{name: "page-size", usage: "Page size. Max value: 1000"},
			{name: "sort-by", usage: "Sort By."},
			{name: "sort-order", usage: "Sort Order. (one of: asc, desc)", enumSet: true, enum: []string{"asc", "desc"}},
		},
		paginated: true,
		resource:  "organizations",
	})
}
