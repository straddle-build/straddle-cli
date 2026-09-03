// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package straddleacct

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

type policyMatrixSpec struct {
	Paths      map[string]policyMatrixPath `json:"paths"`
	Components struct {
		Parameters map[string]policyMatrixParameter `json:"parameters"`
	} `json:"components"`
}

type policyMatrixPath struct {
	Parameters []policyMatrixParameter `json:"parameters"`
	Get        *policyMatrixOperation  `json:"get"`
	Put        *policyMatrixOperation  `json:"put"`
	Post       *policyMatrixOperation  `json:"post"`
	Delete     *policyMatrixOperation  `json:"delete"`
	Options    *policyMatrixOperation  `json:"options"`
	Head       *policyMatrixOperation  `json:"head"`
	Patch      *policyMatrixOperation  `json:"patch"`
	Trace      *policyMatrixOperation  `json:"trace"`
}

type policyMatrixOperation struct {
	Parameters []policyMatrixParameter `json:"parameters"`
}

type policyMatrixParameter struct {
	Ref  string `json:"$ref"`
	Name string `json:"name"`
	In   string `json:"in"`
}

type contractPolicyOperation struct {
	Path                 string
	Method               string
	AcceptsAccountHeader bool
}

type policyIntegrationType struct {
	Name  string
	Value string
	Index int
}

var policyIntegrationTypes = [...]policyIntegrationType{
	{Name: TypeAccount, Value: TypeAccount, Index: 0},
	{Name: TypeSaaS, Value: TypeSaaS, Index: 1},
	{Name: TypeMarketplace, Value: TypeMarketplace, Index: 2},
	{Name: "unset", Value: "", Index: 3},
}

var expectedPolicyMatrix = map[string][4]Decision{
	"GET /v1/accounts/{account_id}":                                  {Forbid, Forbid, Forbid, Forbid},
	"PUT /v1/accounts/{account_id}":                                  {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/accounts":                                               {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/accounts":                                              {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/accounts/{account_id}/onboard":                         {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/accounts/{account_id}/capability_requests":              {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/accounts/{account_id}/capability_requests":             {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/accounts/{account_id}/simulate":                        {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/linked_bank_accounts":                                   {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/linked_bank_accounts":                                  {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/linked_bank_accounts/{linked_bank_account_id}":          {Forbid, Forbid, Forbid, Forbid},
	"PUT /v1/linked_bank_accounts/{linked_bank_account_id}":          {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/linked_bank_accounts/{linked_bank_account_id}/unmask":   {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/organizations":                                          {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/organizations":                                         {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/representatives":                                        {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/representatives":                                       {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/representatives/{representative_id}":                    {Forbid, Forbid, Forbid, Forbid},
	"PUT /v1/representatives/{representative_id}":                    {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/bridge/bank_account":                                   {Forbid, Require, Forbid, Allow},
	"POST /v1/bridge/plaid":                                          {Forbid, Require, Forbid, Allow},
	"POST /v1/bridge/initialize":                                     {Forbid, Require, Forbid, Allow},
	"GET /v1/customers/{id}":                                         {Forbid, Allow, Forbid, Allow},
	"PUT /v1/customers/{id}":                                         {Forbid, Allow, Forbid, Allow},
	"DELETE /v1/customers/{id}":                                      {Forbid, Allow, Forbid, Allow},
	"GET /v1/customers":                                              {Forbid, Allow, Forbid, Allow},
	"POST /v1/customers":                                             {Forbid, Require, Forbid, Allow},
	"GET /v1/customers/{id}/unmasked":                                {Forbid, Allow, Forbid, Allow},
	"GET /v1/customers/{id}/review":                                  {Forbid, Allow, Forbid, Allow},
	"PATCH /v1/customers/{id}/review":                                {Forbid, Allow, Forbid, Allow},
	"GET /v1/paykeys/{id}":                                           {Forbid, Allow, Forbid, Allow},
	"GET /v1/paykeys/{id}/unmasked":                                  {Forbid, Allow, Forbid, Allow},
	"GET /v1/paykeys":                                                {Forbid, Allow, Forbid, Allow},
	"GET /v1/charges/{id}":                                           {Forbid, Allow, Allow, Allow},
	"PUT /v1/charges/{id}":                                           {Forbid, Allow, Allow, Allow},
	"POST /v1/charges":                                               {Forbid, Require, Require, Allow},
	"PUT /v1/charges/{id}/hold":                                      {Forbid, Allow, Allow, Allow},
	"PUT /v1/charges/{id}/release":                                   {Forbid, Allow, Allow, Allow},
	"PUT /v1/charges/{id}/cancel":                                    {Forbid, Allow, Allow, Allow},
	"GET /v1/funding_events":                                         {Forbid, Allow, Allow, Allow},
	"GET /v1/funding_events/{id}":                                    {Forbid, Allow, Allow, Allow},
	"GET /v1/payments":                                               {Forbid, Allow, Allow, Allow},
	"GET /v1/payouts/{id}":                                           {Forbid, Allow, Allow, Allow},
	"PUT /v1/payouts/{id}":                                           {Forbid, Allow, Allow, Allow},
	"POST /v1/payouts":                                               {Forbid, Require, Require, Allow},
	"PUT /v1/payouts/{id}/hold":                                      {Forbid, Allow, Allow, Allow},
	"PUT /v1/payouts/{id}/release":                                   {Forbid, Allow, Allow, Allow},
	"PUT /v1/payouts/{id}/cancel":                                    {Forbid, Allow, Allow, Allow},
	"GET /v1/organizations/{organization_id}":                        {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/representatives/{representative_id}/unmask":             {Forbid, Forbid, Forbid, Forbid},
	"GET /v1/paykeys/{id}/reveal":                                    {Forbid, Allow, Forbid, Allow},
	"PUT /v1/customers/{id}/refresh_review":                          {Forbid, Allow, Forbid, Allow},
	"GET /v1/charges/{id}/unmask":                                    {Forbid, Allow, Allow, Allow},
	"GET /v1/payouts/{id}/unmask":                                    {Forbid, Allow, Allow, Allow},
	"PUT /v1/paykeys/{id}/cancel":                                    {Forbid, Allow, Forbid, Allow},
	"POST /v1/bridge/quiltt":                                         {Forbid, Require, Forbid, Allow},
	"GET /v1/paykeys/{id}/review":                                    {Forbid, Allow, Forbid, Allow},
	"PATCH /v1/paykeys/{id}/review":                                  {Forbid, Allow, Forbid, Allow},
	"PATCH /v1/linked_bank_accounts/{linked_bank_account_id}/cancel": {Forbid, Forbid, Forbid, Forbid},
	"PUT /v1/paykeys/{id}/refresh_review":                            {Forbid, Allow, Forbid, Allow},
	"PUT /v1/paykeys/{id}/refresh_balance":                           {Forbid, Allow, Forbid, Allow},
	"POST /v1/funding_events/simulate":                               {Forbid, Allow, Allow, Allow},
	"PATCH /v1/paykeys/{id}/unblock":                                 {Forbid, Allow, Forbid, Allow},
	"GET /v1/account_settings/{account_id}":                          {Forbid, Forbid, Forbid, Forbid},
	"POST /v1/charges/{id}/resubmit":                                 {Forbid, Require, Require, Allow},
	"POST /v1/charges/{id}/refund":                                   {Forbid, Require, Require, Allow},
	"GET /v1/funding_event_payments/{id}":                            {Forbid, Allow, Allow, Allow},
	"POST /v1/payouts/{id}/resubmit":                                 {Forbid, Require, Require, Allow},
	"POST /v1/charges/{id}/authorization":                            {Forbid, Require, Require, Allow},
	"POST /v1/payouts/{id}/authorization":                            {Forbid, Require, Require, Allow},
}

var knownDivergences = map[string]string{}

func TestPolicyMatrix(t *testing.T) {
	contractOperations := loadPolicyMatrixContract(t)
	if got := len(expectedPolicyMatrix); got != 70 {
		t.Fatalf("expected policy matrix has %d operations, want 70", got)
	}
	if got := len(contractOperations); got != 70 {
		t.Fatalf("contract has %d operations, want 70", got)
	}

	keys := make([]string, 0, len(contractOperations))
	for key := range contractOperations {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	observedDivergences := make(map[string][]string)
	for _, key := range keys {
		operation := contractOperations[key]
		expected, ok := expectedPolicyMatrix[key]
		if !ok {
			t.Errorf("contract operation %q is missing from expected policy matrix", key)
			continue
		}

		for _, integrationType := range policyIntegrationTypes {
			t.Run(key+"/"+integrationType.Name, func(t *testing.T) {
				got := Classify(operation.Path, operation.Method, integrationType.Value)
				if want := expected[integrationType.Index]; got != want {
					t.Errorf("Classify(%q, %q, %q) = %v, want %v", operation.Path, operation.Method, integrationType.Value, got, want)
				}
			})
		}

		if !operation.AcceptsAccountHeader {
			for _, integrationType := range policyIntegrationTypes {
				if got := Classify(operation.Path, operation.Method, integrationType.Value); got != Forbid {
					observedDivergences[key] = append(observedDivergences[key], fmt.Sprintf("contract omits %s but %s policy returns %v", Header, integrationType.Name, got))
				}
			}
			continue
		}

		saasDecision := Classify(operation.Path, operation.Method, TypeSaaS)
		if saasDecision != Require && saasDecision != Allow {
			observedDivergences[key] = append(observedDivergences[key], fmt.Sprintf("contract declares %s but saas policy returns %v", Header, saasDecision))
		}

		marketplaceDecision := Classify(operation.Path, operation.Method, TypeMarketplace)
		marketplaceCustomerOwned := customerOwnedResources[ResourceFromPath(operation.Path)]
		if marketplaceCustomerOwned && marketplaceDecision != Forbid {
			observedDivergences[key] = append(observedDivergences[key], fmt.Sprintf("marketplace customer-owned policy returns %v, want %v", marketplaceDecision, Forbid))
		}
		if !marketplaceCustomerOwned && marketplaceDecision != Require && marketplaceDecision != Allow {
			observedDivergences[key] = append(observedDivergences[key], fmt.Sprintf("contract declares %s but marketplace policy returns %v", Header, marketplaceDecision))
		}
	}

	for key := range expectedPolicyMatrix {
		if _, ok := contractOperations[key]; !ok {
			t.Errorf("expected policy matrix operation %q is absent from contract", key)
		}
	}

	assertKnownDivergences(t, observedDivergences)
}

func loadPolicyMatrixContract(t *testing.T) map[string]contractPolicyOperation {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "spec.yaml"))
	if err != nil {
		t.Fatalf("read spec.yaml: %v", err)
	}

	var spec policyMatrixSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse spec.yaml: %v", err)
	}

	operations := make(map[string]contractPolicyOperation)
	for path, pathItem := range spec.Paths {
		pathDeclaresHeader, err := policyMatrixDeclaresAccountHeader(pathItem.Parameters, spec.Components.Parameters)
		if err != nil {
			t.Fatalf("parse parameters for path %q: %v", path, err)
		}

		methods := [...]struct {
			name      string
			operation *policyMatrixOperation
		}{
			{name: "GET", operation: pathItem.Get},
			{name: "PUT", operation: pathItem.Put},
			{name: "POST", operation: pathItem.Post},
			{name: "DELETE", operation: pathItem.Delete},
			{name: "OPTIONS", operation: pathItem.Options},
			{name: "HEAD", operation: pathItem.Head},
			{name: "PATCH", operation: pathItem.Patch},
			{name: "TRACE", operation: pathItem.Trace},
		}
		for _, method := range methods {
			if method.operation == nil {
				continue
			}
			operationDeclaresHeader, err := policyMatrixDeclaresAccountHeader(method.operation.Parameters, spec.Components.Parameters)
			if err != nil {
				t.Fatalf("parse parameters for %s %s: %v", method.name, path, err)
			}
			key := method.name + " " + path
			if _, exists := operations[key]; exists {
				t.Fatalf("duplicate contract operation %q", key)
			}
			operations[key] = contractPolicyOperation{
				Path:                 path,
				Method:               method.name,
				AcceptsAccountHeader: pathDeclaresHeader || operationDeclaresHeader,
			}
		}
	}
	return operations
}

func policyMatrixDeclaresAccountHeader(parameters []policyMatrixParameter, components map[string]policyMatrixParameter) (bool, error) {
	const componentParameterPrefix = "#/components/parameters/"

	for _, parameter := range parameters {
		resolved := parameter
		if parameter.Ref != "" {
			if !strings.HasPrefix(parameter.Ref, componentParameterPrefix) {
				return false, fmt.Errorf("unsupported parameter reference %q", parameter.Ref)
			}
			name := strings.TrimPrefix(parameter.Ref, componentParameterPrefix)
			component, ok := components[name]
			if !ok {
				return false, fmt.Errorf("parameter reference %q is undefined", parameter.Ref)
			}
			resolved = component
		}
		if resolved.In == "header" && resolved.Name == Header {
			return true, nil
		}
	}
	return false, nil
}

func assertKnownDivergences(t *testing.T, observed map[string][]string) {
	t.Helper()

	for key, reasons := range observed {
		if _, ok := knownDivergences[key]; !ok {
			t.Errorf("unexpected contract divergence %q: %s", key, strings.Join(reasons, "; "))
		}
	}
	for key, reason := range knownDivergences {
		if reason == "" || strings.Contains(reason, "\n") {
			t.Errorf("known divergence %q must have a one-line reason", key)
		}
		if _, ok := observed[key]; !ok {
			t.Errorf("known contract divergence %q is no longer observed: %s", key, reason)
		}
	}
}
