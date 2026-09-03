// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package straddleacct

import "testing"

func TestResourceFromPath(t *testing.T) {
	cases := map[string]string{
		"/v1/charges":                           "charges",
		"/v1/charges/{id}/hold":                 "charges",
		"/v1/funding_event_payments/{id}":       "funding_event_payments",
		"/v1/reports/total_customers_by_status": "reports",
		"/v1/accounts/{account_id}/onboard":     "accounts",
		"":                                      "",
	}
	for path, want := range cases {
		if got := ResourceFromPath(path); got != want {
			t.Errorf("ResourceFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestFallbackAcceptsHeader(t *testing.T) {
	cases := map[string]bool{
		"/v1/charges":                           true,
		"/v1/payouts":                           true,
		"/v1/customers":                         true,
		"/v1/paykeys":                           true,
		"/v1/bridge/initialize":                 true,
		"/v1/funding_events":                    true,
		"/v1/funding_event_payments/{id}":       true,
		"/v1/payments":                          true,
		"/v1/reports/total_customers_by_status": true,
		"/v1/accounts":                          false,
		"/v1/representatives":                   false,
		"":                                      false,
	}
	for path, want := range cases {
		if got := FallbackAcceptsHeader(path); got != want {
			t.Errorf("FallbackAcceptsHeader(%q) = %t, want %t", path, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		path, method, itype string
		acceptsHeader       bool
		want                Decision
	}{
		{"/v1/charges", "POST", TypeAccount, true, Forbid},
		{"/v1/customers", "GET", TypeAccount, true, Forbid},
		{"/v1/charges", "POST", TypeSaaS, true, Require},
		{"/v1/payouts", "POST", TypeSaaS, true, Require},
		{"/v1/customers", "POST", TypeSaaS, true, Require},
		{"/v1/bridge/initialize", "POST", TypeSaaS, true, Require},
		{"/v1/charges/{id}", "GET", TypeSaaS, true, Allow},
		{"/v1/customers", "GET", TypeSaaS, true, Allow},
		{"/v1/funding_events", "GET", TypeSaaS, true, Allow},
		{"/v1/reports/total_customers_by_status", "POST", TypeSaaS, true, Allow},
		{"/v1/accounts", "POST", TypeSaaS, false, Forbid},
		{"/v1/representatives", "POST", TypeSaaS, false, Forbid},
		{"/v1/charges", "POST", TypeMarketplace, true, Require},
		{"/v1/payouts", "POST", TypeMarketplace, true, Require},
		{"/v1/customers", "POST", TypeMarketplace, true, Forbid},
		{"/v1/customers", "GET", TypeMarketplace, true, Forbid},
		{"/v1/paykeys", "GET", TypeMarketplace, true, Forbid},
		{"/v1/bridge/initialize", "POST", TypeMarketplace, true, Forbid},
		{"/v1/reports/total_customers_by_status", "POST", TypeMarketplace, true, Forbid},
		{"/v1/funding_events", "GET", TypeMarketplace, true, Allow},
		{"/v1/payments", "GET", TypeMarketplace, true, Allow},
		{"/v1/charges/{id}/hold", "PUT", TypeMarketplace, true, Allow},
		{"/v1/accounts", "POST", TypeMarketplace, false, Forbid},
		{"/v1/charges", "POST", "", true, Allow},
		{"/v1/accounts", "POST", "", false, Forbid},
		{"/v1/charges", "POST", TypeSaaS, false, Forbid},
		{"/v1/new_contract_resource", "GET", TypeSaaS, true, Allow},
	}
	for _, tc := range cases {
		got := Classify(tc.path, tc.method, tc.itype, tc.acceptsHeader)
		if got != tc.want {
			t.Errorf("Classify(%q,%q,%q,%t) = %v, want %v", tc.path, tc.method, tc.itype, tc.acceptsHeader, got, tc.want)
		}
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name        string
		d           Decision
		flagAccount string
		flagChanged bool
		sticky      string
		wantValue   string
		wantSend    bool
		wantErr     bool
	}{
		{"forbid+explicit flag errors", Forbid, "acct_x", true, "", "", false, true},
		{"forbid+sticky silently omits", Forbid, "", false, "acct_sticky", "", false, false},
		{"forbid+nothing omits", Forbid, "", false, "", "", false, false},
		{"require+flag sends", Require, "acct_x", true, "", "acct_x", true, false},
		{"require+sticky sends", Require, "", false, "acct_sticky", "acct_sticky", true, false},
		{"require+nothing errors", Require, "", false, "", "", false, true},
		{"require flag overrides sticky", Require, "acct_flag", true, "acct_sticky", "acct_flag", true, false},
		{"allow+flag sends", Allow, "acct_x", true, "", "acct_x", true, false},
		{"allow+sticky sends", Allow, "", false, "acct_sticky", "acct_sticky", true, false},
		{"allow+nothing omits", Allow, "", false, "", "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, send, err := Resolve(tc.d, tc.flagAccount, tc.flagChanged, tc.sticky)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if val != tc.wantValue {
				t.Errorf("value = %q, want %q", val, tc.wantValue)
			}
			if send != tc.wantSend {
				t.Errorf("send = %v, want %v", send, tc.wantSend)
			}
		})
	}
}
