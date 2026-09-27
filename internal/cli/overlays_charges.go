// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

func init() {
	registerCommandOverlay("charges.cancel", commandOverlay{
		short:   "Cancel a charge to prevent it from being originated for processing. The status of the charge must be `created`,...",
		long:    "",
		example: "  straddle charges cancel charge 550e8400-e29b-41d4-a716-446655440000",
		aliases: []string{"charge", "update"},
		flags: []flagOverlay{
			{name: "reason", usage: "Details about why the charge status was updated."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "cancel",
	})
	registerCommandOverlay("charges.create", commandOverlay{
		short:   "Use charges to collect money from a customer for the sale of goods or services.",
		long:    "",
		example: "  straddle charges create --amount 1250 --config-balance-check enabled --consent-type internet --currency USD --description \"Invoice 1001\" --device-ip-address 192.0.2.1 --external-id invoice-1001 --paykey 550e8400-e29b-41d4-a716-446655440000 --payment-date 2026-01-15",
		flags: []flagOverlay{
			{name: "amount", usage: "The amount of the charge in cents."},
			{name: "config-auto-hold", usage: "Defines whether to automatically place this charge on hold after being created."},
			{name: "config-auto-hold-message", usage: "The reason the charge is being automatically held on creation."},
			{name: "config-balance-check", usage: "Defines whether to check the customer's balance before processing the charge. Accepted values: required, enabled, disabled."},
			{name: "consent-type", usage: "The channel or mechanism through which the payment was authorized. Use `internet` for payments made online or..."},
			{name: "currency", usage: "The currency of the charge. Only USD is supported."},
			{name: "description", usage: "An arbitrary description for the charge."},
			{name: "device-ip-address", usage: "The IP address of the device used when the customer authorized the charge or payout. Use `0.0.0.0` to represent an..."},
			{name: "external-id", usage: "Unique identifier for the charge in your database. This value must be unique across all charges."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the charge in a..."},
			{name: "paykey", usage: "Value of the `paykey` used for the charge."},
			{name: "payment-date", usage: "The desired date on which the payment should be occur. For charges, this means the date you want the customer to be..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "charges",
	})
	registerCommandOverlay("charges.get", commandOverlay{
		short:    "Retrieves the details of an existing charge. Supply the unique charge `id`, and Straddle will return the...",
		long:     "",
		example:  "  straddle charges get 550e8400-e29b-41d4-a716-446655440000",
		resource: "charges",
	})
	registerCommandOverlay("charges.get-unmasked-charge", commandOverlay{
		short:    "Get a charge by id.",
		long:     "",
		example:  "  straddle charges unmask charges-v1-get 550e8400-e29b-41d4-a716-446655440000",
		aliases:  []string{"charges-v1-get", "get"},
		resource: "unmask",
	})
	registerCommandOverlay("charges.hold", commandOverlay{
		short:   "Place a charge on hold to prevent it from being originated for processing. The status of the charge must be...",
		long:    "",
		example: "  straddle charges hold charge 550e8400-e29b-41d4-a716-446655440000",
		aliases: []string{"charge", "update"},
		flags: []flagOverlay{
			{name: "reason", usage: "Details about why the charge status was updated."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "hold",
	})
	registerCommandOverlay("charges.refund", commandOverlay{
		short:    "Refund a paid charge",
		long:     "",
		example:  "  straddle charges refund <id>",
		resource: "charges",
		action:   "refund",
	})
	registerCommandOverlay("charges.release", commandOverlay{
		short:   "Release a charge from an `on_hold` status to allow it to be rescheduled for processing.",
		long:    "",
		example: "  straddle charges release charge 550e8400-e29b-41d4-a716-446655440000",
		aliases: []string{"charge", "update"},
		flags: []flagOverlay{
			{name: "reason", usage: "Details about why the charge status was updated."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "release",
	})
	registerCommandOverlay("charges.resubmit", commandOverlay{
		short:   "Resubmit a failed or reversed charge.",
		long:    "",
		example: "  straddle charges resubmit create 550e8400-e29b-41d4-a716-446655440000",
		flags: []flagOverlay{
			{name: "description", usage: "Description."},
			{name: "external-id", usage: "External id."},
			{name: "payment-date", usage: "Payment date."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "resubmit",
	})
	registerCommandOverlay("charges.update", commandOverlay{
		short:   "Change the values of parameters associated with a charge prior to processing. The status of the charge must be...",
		long:    "",
		example: "  straddle charges update 550e8400-e29b-41d4-a716-446655440000 --description example-value",
		flags: []flagOverlay{
			{name: "amount", usage: "The amount of the charge in cents."},
			{name: "description", usage: "An arbitrary description for the charge."},
			{name: "metadata", usage: "Up to 20 additional user-defined key-value pairs. Useful for storing additional information about the charge in a..."},
			{name: "payment-date", usage: "The desired date on which the payment should be occur. For charges, this means the date you want the customer to be..."},
			{name: "stdin", usage: "Read request body as JSON from stdin"},
		},
		resource: "charges",
	})
	registerCommandOverlay("charges.upload-charge-authorization-proof", commandOverlay{
		short:   "Upload a proof-of-authorization document for a charge",
		long:    "Uploads a proof-of-authorization document for a charge: a PDF, PNG, JPEG, DOC, or DOCX file of at most 10 MiB whose contents match its extension. The file is streamed, never loaded into memory. A later upload adds another document and does not replace an existing one.",
		example: "  straddle charges upload-authorization-proof 550e8400-e29b-41d4-a716-446655440000 --file ./authorization.pdf",
		flags: []flagOverlay{
			{name: "file", usage: "Path to the document to upload (.pdf, .png, .jpg, .jpeg, .doc, or .docx; at most 10 MiB).", maxBytes: 10 << 20},
		},
		resource: "charges",
		action:   "upload-authorization-proof",
	})
}
