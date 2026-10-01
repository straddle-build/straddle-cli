// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// The sandbox reference is curated by hand, so the pinned contract is the
// check that catches a contract sync adding or removing an outcome.
func TestSandboxOutcomesMatchContractEnums(t *testing.T) {
	raw, err := os.ReadFile("../../spec.yaml")
	if err != nil {
		t.Fatalf("read spec.yaml: %v", err)
	}
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Enum []string `json:"enum"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse spec.yaml: %v", err)
	}

	stdout, stderr, err := runRootForAPITest(t, []string{"sandbox", "outcomes", "--json"}, "")
	if err != nil {
		t.Fatalf("sandbox outcomes --json: %v\nstderr: %s", err, stderr)
	}
	var got map[string][]sandboxOutcome
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout)
	}

	for _, tc := range []struct {
		section string
		schema  string
	}{
		{"customers", "SimulatedCustomerOutcome"},
		{"paykeys", "SimulatedPaykeyOutcome"},
		{"charges_payouts", "SimulatedPaymentOutcome"},
	} {
		t.Run(tc.section, func(t *testing.T) {
			want := slices.Sorted(slices.Values(spec.Components.Schemas[tc.schema].Enum))
			if len(want) == 0 {
				t.Fatalf("spec.yaml has no enum for %s", tc.schema)
			}
			var values []string
			for _, o := range got[tc.section] {
				values = append(values, o.Value)
			}
			slices.Sort(values)
			if !slices.Equal(values, want) {
				t.Fatalf("straddle sandbox %s = %v, contract %s = %v", tc.section, values, tc.schema, want)
			}
		})
	}
}

// Straddle's rule: every return code except R01 and R09 blocks the paykey,
// whether the charge ended failed or reversed.
func TestSandboxReturnOutcomesStatePaykeyBlock(t *testing.T) {
	stdout, stderr, err := runRootForAPITest(t, []string{"sandbox", "outcomes", "--json"}, "")
	if err != nil {
		t.Fatalf("sandbox outcomes --json: %v\nstderr: %s", err, stderr)
	}
	var got map[string][]sandboxOutcome
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout)
	}

	for _, o := range got["charges_payouts"] {
		isReturn := strings.HasPrefix(o.Value, "failed_") || strings.HasPrefix(o.Value, "reversed_")
		if isReturn && o.Code == "" {
			t.Errorf("return outcome %s has no return code", o.Value)
			continue
		}
		if o.Code == "" {
			continue
		}
		want := "blocks the paykey"
		if o.Code == "R01" || o.Code == "R09" {
			want = "doesn't block the paykey"
		}
		if !strings.Contains(o.Description, want) {
			t.Errorf("outcome %s (%s) description %q should say %q", o.Value, o.Code, o.Description, want)
		}
	}
}
