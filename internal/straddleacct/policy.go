// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
//
// Package straddleacct decides when a Straddle API call must carry the
// Straddle-Account-Id header, based on the operation and the platform's
// integration type. It is the policy brain for the CLI command path.
//
// Generated commands supply header capability from their registered contract
// surface. Operations without a registered surface fall back to the legacy
// resource-level capability through FallbackAcceptsHeader. The
// integration-type overlay is business policy the contract cannot encode.
// See https://docs.straddle.com/guides/embed/api-headers.
package straddleacct

import "strings"

// Decision is the header policy for one operation under one integration type.
type Decision int

const (
	// Forbid never sends the header (and rejects an explicit --account).
	Forbid Decision = iota
	// Require makes the account mandatory; a missing value is an error.
	Require
	// Allow sends the header when a value is available, else omits it.
	Allow
)

// Integration types.
const (
	TypeAccount     = "account"
	TypeSaaS        = "saas"
	TypeMarketplace = "marketplace"
)

// Header is the HTTP header that scopes a platform call to an embedded account.
const Header = "Straddle-Account-Id"

// fallbackHeaderCapableResources preserves the resource-level behavior for
// operations without a registered contract surface.
var fallbackHeaderCapableResources = map[string]bool{
	"charges":                true,
	"payouts":                true,
	"customers":              true,
	"paykeys":                true,
	"bridge":                 true,
	"funding_events":         true,
	"funding_event_payments": true,
	"payments":               true,
	"reports":                true,
}

// customerOwnedResources belong to the platform itself under the marketplace
// model (the marketplace's own users), so a marketplace never scopes them to
// an embedded account. Under SaaS the embedded account owns them, so they are
// scoped normally. reports/total_customers_by_status is customer-category.
var customerOwnedResources = map[string]bool{
	"customers": true,
	"paykeys":   true,
	"bridge":    true,
	"reports":   true,
}

// saasCreateScoped names the resources a SaaS platform must scope to an
// embedded account at creation time (the API rejects platform-level customers
// for SaaS, and money movement is always account-attributed).
var saasCreateScoped = map[string]bool{
	"charges":   true,
	"payouts":   true,
	"customers": true,
	"paykeys":   true,
	"bridge":    true,
}

// ResourceFromPath extracts the resource segment from a straddle:path annotation
// value such as "/v1/charges/{id}/hold" -> "charges". Returns "" when not derivable.
func ResourceFromPath(ppPath string) string {
	parts := strings.Split(strings.Trim(ppPath, "/"), "/")
	for i, p := range parts {
		if p == "v1" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// FallbackAcceptsHeader reports the legacy resource-level capability for an
// operation without a registered contract surface.
func FallbackAcceptsHeader(ppPath string) bool {
	return fallbackHeaderCapableResources[ResourceFromPath(ppPath)]
}

// Classify returns the header policy for an operation under an integration
// type. acceptsHeader is the operation capability declared by the contract.
// An unset/unknown integration type sends the header only when explicitly
// supplied and never requires it.
func Classify(ppPath, ppMethod, integrationType string, acceptsHeader bool) Decision {
	if !acceptsHeader {
		return Forbid
	}
	resource := ResourceFromPath(ppPath)
	isPost := strings.EqualFold(ppMethod, "POST")
	switch integrationType {
	case TypeAccount:
		return Forbid
	case TypeMarketplace:
		if customerOwnedResources[resource] {
			return Forbid
		}
		if isPost && (resource == "charges" || resource == "payouts") {
			return Require
		}
		return Allow
	case TypeSaaS:
		if isPost && saasCreateScoped[resource] {
			return Require
		}
		return Allow
	default:
		return Allow
	}
}

// PolicyError is returned by Resolve when the decision and the available
// account value conflict. Reason is "required" or "forbidden".
type PolicyError struct {
	Reason  string
	Message string
}

func (e *PolicyError) Error() string { return e.Message }

// Resolve applies a Decision to the available account values and returns the
// header value to send. The per-call flag overrides the sticky account
// whenever the flag was supplied.
func Resolve(d Decision, flagAccount string, flagChanged bool, sticky string) (value string, send bool, err error) {
	effective := sticky
	if flagChanged {
		effective = flagAccount
	}
	switch d {
	case Forbid:
		if flagChanged && flagAccount != "" {
			return "", false, &PolicyError{
				Reason:  "forbidden",
				Message: "this operation does not act on behalf of an embedded account; remove --account",
			}
		}
		return "", false, nil
	case Require:
		if effective == "" {
			return "", false, &PolicyError{
				Reason:  "required",
				Message: "this operation acts on behalf of an embedded account; an account id is required",
			}
		}
		return effective, true, nil
	default: // Allow
		if effective == "" {
			return "", false, nil
		}
		return effective, true, nil
	}
}
