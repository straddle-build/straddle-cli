// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package apisync_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/straddle-build/cli/internal/apisync"
	"sigs.k8s.io/yaml"
)

func TestCurrentSpecOperationsAreCoveredByCheckedInAnnotations(t *testing.T) {
	t.Parallel()

	repo := testRepoRoot(t)
	result, err := apisync.CheckSpecAgainstRepo(filepath.Join(repo, "spec.yaml"), repo)
	if err != nil {
		t.Fatalf("CheckSpecAgainstRepo: %v", err)
	}
	allowReviewDrift := os.Getenv("STRADDLE_API_SYNC_REVIEW") == "true"
	if !coverageAccepted(result, allowReviewDrift) {
		t.Fatalf("current spec coverage failed: missing=%d extra=%d duplicate=%d invalid=%d operation_id_mismatch=%d", len(result.Missing), len(result.Extra), len(result.DuplicateAnnotations), len(result.InvalidAnnotations), len(result.OperationIDMismatches))
	}
	if !unsupportedInventoryAccepted(len(result.UnsupportedOperations), allowReviewDrift) {
		t.Fatalf("unsupported operations = %d, want two multipart upload operations", len(result.UnsupportedOperations))
	}
}

func TestAPISyncReviewModeAllowsOnlyRemovedOrRenamedOperations(t *testing.T) {
	t.Parallel()

	reviewable := apisync.CheckResult{
		Extra:                 []apisync.Annotation{{File: "removed.go"}},
		OperationIDMismatches: []apisync.OperationIDMismatch{{Key: "GET /v1/widgets"}},
	}
	if coverageAccepted(reviewable, false) {
		t.Fatal("strict coverage accepted review-only drift")
	}
	if !coverageAccepted(reviewable, true) {
		t.Fatal("API sync review mode rejected removed or renamed operations")
	}
	reviewable.Missing = []apisync.Operation{{Key: "POST /v1/widgets"}}
	if coverageAccepted(reviewable, true) {
		t.Fatal("API sync review mode accepted a missing supported operation")
	}
	if unsupportedInventoryAccepted(3, false) {
		t.Fatal("strict coverage accepted an unexpected unsupported operation")
	}
	if !unsupportedInventoryAccepted(3, true) {
		t.Fatal("API sync review mode rejected an unsupported operation for human review")
	}
}

func coverageAccepted(result apisync.CheckResult, allowReviewDrift bool) bool {
	if result.HasBlockingIssues() {
		return false
	}
	return allowReviewDrift || result.OK
}

func unsupportedInventoryAccepted(count int, allowReviewDrift bool) bool {
	return allowReviewDrift || count == 2
}

func TestParseSpecLoadsYAMLAndResolvesSharedParameters(t *testing.T) {
	t.Parallel()

	operations, err := apisync.ParseSpec([]byte(`
openapi: 3.1.0
info: {version: 1.2.3}
paths:
  /v1/widgets/{id}:
    get:
      tags: [widgets]
      operationId: getWidget
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
        - {$ref: '#/components/parameters/RequestId'}
components:
  parameters:
    RequestId:
      name: Request-Id
      in: header
      schema: {type: string}
`))
	if err != nil {
		t.Fatalf("ParseSpec: %v", err)
	}
	if len(operations) != 1 {
		t.Fatalf("operations = %d, want 1", len(operations))
	}
	operation := operations[0]
	if operation.OperationID != "getWidget" || len(operation.PathParameters) != 1 || len(operation.HeaderParameters) != 1 {
		t.Fatalf("operation = %#v", operation)
	}
	if operation.HeaderParameters[0].Name != "Request-Id" {
		t.Fatalf("header parameter = %#v, want resolved Request-Id", operation.HeaderParameters[0])
	}
}

func TestCheckCoverageUsesOperationIDsAndIgnoresInternalCommands(t *testing.T) {
	t.Parallel()

	operations := []apisync.Operation{{Key: "GET /v1/widgets", OperationID: "getWidget", Method: "GET", Path: "/v1/widgets"}}
	inventory := apisync.Inventory{Annotations: []apisync.Annotation{
		{OperationID: "wrongOperation", Method: "GET", Path: "/v1/widgets", File: "widgets.go"},
		{Method: "POST", Path: "/v1/internal", File: "internal.go", Internal: true},
	}}
	result := apisync.CheckCoverage(operations, inventory)
	if result.OK || len(result.OperationIDMismatches) != 1 {
		t.Fatalf("result = %#v, want one operationId mismatch", result)
	}
	if len(result.Extra) != 0 || result.AnnotatedEndpoints != 1 {
		t.Fatalf("internal command affected contract coverage: %#v", result)
	}
}

func TestAPISyncWorkflowAlwaysProposesNewPublishedContracts(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, filepath.Join(testRepoRoot(t), ".github", "workflows", "api-sync.yml"))
	if len(workflow.On.Schedule) != 1 {
		t.Fatalf("schedule triggers = %d, want 1", len(workflow.On.Schedule))
	}
	if got := workflow.On.RepositoryDispatch.Types; len(got) != 1 || got[0] != "straddle-contract-published" {
		t.Fatalf("repository dispatch types = %#v", got)
	}
	versionInput := workflow.On.WorkflowDispatch.Inputs["contract_version"]
	if !versionInput.Required || versionInput.Type != "string" {
		t.Fatalf("contract_version input = %#v, want required string", versionInput)
	}

	steps := stepsByName(workflow.Jobs["sync"])
	if steps["Update pinned contract"].If != "steps.contract.outputs.status == 'new'" {
		t.Fatalf("update condition = %q", steps["Update pinned contract"].If)
	}
	generateCondition := steps["Generate supported endpoint additions"].If
	if generateCondition != "steps.contract.outputs.status == 'new' && steps.drift.outputs.supported_additions != '0'" {
		t.Fatalf("generation condition = %q", generateCondition)
	}
	openPR := steps["Open contract synchronization pull request"]
	if openPR.Uses != "peter-evans/create-pull-request@v8" {
		t.Fatalf("PR action = %q", openPR.Uses)
	}
	if openPR.If != "steps.contract.outputs.status == 'new' && env.DRY_RUN != 'true' && env.HAS_API_SYNC_BOT_TOKEN == 'true'" {
		t.Fatalf("PR condition = %q", openPR.If)
	}
	branch, ok := openPR.With["branch"].(string)
	if !ok || !strings.Contains(branch, "steps.release.outputs.version") {
		t.Fatalf("PR branch = %#v, want a contract-version-specific branch", openPR.With["branch"])
	}
	if !strings.Contains(steps["Verify synchronized CLI"].Run, "--review-drift") {
		t.Fatal("post-generation verification would block reviewable removals or renames")
	}
	if !strings.Contains(steps["Verify synchronized CLI"].Run, "STRADDLE_API_SYNC_REVIEW=true go test ./...") {
		t.Fatal("API sync tests would apply strict coverage before opening the review PR")
	}
	fetchRun := steps["Fetch and verify exact Scalar contract"].Run
	for _, required := range []string{"git ls-remote", "branch_lookup_status", "case", "exit", "pending-contract.lock.json", "verify-lock --lock", "candidate-status --lock"} {
		if !strings.Contains(fetchRun, required) {
			t.Fatalf("pending contract branch verification is missing %q", required)
		}
	}
	for _, step := range workflow.Jobs["sync"].Steps {
		if step.Name == "Queue generated PR for auto-merge" {
			t.Fatal("API sync workflow still auto-merges generated changes")
		}
	}
}

func TestMergedContractChangeTriggersExistingDistributionWorkflow(t *testing.T) {
	t.Parallel()

	repo := testRepoRoot(t)
	tagWorkflow := readWorkflow(t, filepath.Join(repo, ".github", "workflows", "api-sync-release.yml"))
	if got := tagWorkflow.On.Push.Branches; len(got) != 1 || got[0] != "main" {
		t.Fatalf("release handoff branches = %#v", got)
	}
	if got := tagWorkflow.On.Push.Paths; len(got) != 1 || got[0] != "contract.lock.json" {
		t.Fatalf("release handoff paths = %#v", got)
	}
	tagJob := tagWorkflow.Jobs["tag"]
	steps := stepsByName(tagJob)
	if steps["Create patch release tag"].If != "steps.version.outputs.already_tagged != 'true'" {
		t.Fatalf("tag creation condition = %q", steps["Create patch release tag"].If)
	}
	createTagRun := steps["Create patch release tag"].Run
	if !strings.Contains(createTagRun, "git push origin") || strings.Contains(createTagRun, "gh api") {
		t.Fatalf("tag creation must push the tag so the release push trigger runs:\n%s", createTagRun)
	}

	releaseWorkflow := readWorkflow(t, filepath.Join(repo, ".github", "workflows", "release.yml"))
	if got := releaseWorkflow.On.Push.Tags; len(got) != 1 || got[0] != "v*" {
		t.Fatalf("release tag triggers = %#v", got)
	}
}

func TestPullRequestCIRejectsContractDowngrades(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, filepath.Join(testRepoRoot(t), ".github", "workflows", "ci.yml"))
	steps := stepsByName(workflow.Jobs["validate"])
	if !strings.Contains(steps["Verify API contract lock"].Run, "verify-lock") {
		t.Fatal("CI does not verify the proposed contract lock")
	}
	downgradeCheck := steps["Reject API contract downgrades"]
	if downgradeCheck.If != "github.event_name == 'pull_request'" {
		t.Fatalf("downgrade check condition = %q", downgradeCheck.If)
	}
	if !strings.Contains(downgradeCheck.Env["BASE_SHA"], "pull_request.base.sha") {
		t.Fatalf("downgrade check base SHA = %q", downgradeCheck.Env["BASE_SHA"])
	}
	for _, required := range []string{"base-contract.lock.json", "base-spec.yaml", "candidate-status"} {
		if !strings.Contains(downgradeCheck.Run, required) {
			t.Fatalf("downgrade check is missing %q", required)
		}
	}
}

type workflowDocument struct {
	On struct {
		Schedule []struct {
			Cron string `json:"cron"`
		} `json:"schedule"`
		WorkflowDispatch struct {
			Inputs map[string]struct {
				Required bool   `json:"required"`
				Type     string `json:"type"`
			} `json:"inputs"`
		} `json:"workflow_dispatch"`
		RepositoryDispatch struct {
			Types []string `json:"types"`
		} `json:"repository_dispatch"`
		PullRequest struct {
			Types []string `json:"types"`
		} `json:"pull_request"`
		Push struct {
			Tags     []string `json:"tags"`
			Branches []string `json:"branches"`
			Paths    []string `json:"paths"`
		} `json:"push"`
	} `json:"on"`
	Jobs map[string]workflowJob `json:"jobs"`
}

type workflowJob struct {
	If    string         `json:"if"`
	Steps []workflowStep `json:"steps"`
}

type workflowStep struct {
	Name string            `json:"name"`
	If   string            `json:"if"`
	Uses string            `json:"uses"`
	Run  string            `json:"run"`
	Env  map[string]string `json:"env"`
	With map[string]any    `json:"with"`
}

func readWorkflow(t *testing.T, path string) workflowDocument {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow %s: %v", path, err)
	}
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		t.Fatalf("parse workflow %s: %v", path, err)
	}
	var workflow workflowDocument
	if err := json.Unmarshal(jsonData, &workflow); err != nil {
		t.Fatalf("decode workflow %s: %v", path, err)
	}
	return workflow
}

func stepsByName(job workflowJob) map[string]workflowStep {
	steps := make(map[string]workflowStep, len(job.Steps))
	for _, step := range job.Steps {
		steps[step.Name] = step
	}
	return steps
}

func TestClassifyDriftReportsUnsupportedNonJSONRequestBodyAddition(t *testing.T) {
	t.Parallel()

	baseSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)
	headSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/uploads": {
				"post": {
					"tags": ["Uploads"],
					"operationId": "CreateUpload",
					"summary": "Upload a file",
					"requestBody": {
						"required": true,
						"content": {
							"multipart/form-data": {}
						}
					}
				}
			},
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)

	baseOps, err := apisync.LoadSpec(baseSpec)
	if err != nil {
		t.Fatalf("LoadSpec(base): %v", err)
	}
	headOps, err := apisync.LoadSpec(headSpec)
	if err != nil {
		t.Fatalf("LoadSpec(head): %v", err)
	}

	result := apisync.ClassifyDrift(baseOps, headOps)
	if result.NoDrift {
		t.Fatalf("NoDrift = true, want unsupported addition to be reported")
	}
	if len(result.SupportedAdditions) != 0 {
		t.Fatalf("SupportedAdditions = %d, want 0 for non-JSON request body", len(result.SupportedAdditions))
	}
	if len(result.UnsupportedOperations) != 1 {
		t.Fatalf("UnsupportedOperations = %d, want 1", len(result.UnsupportedOperations))
	}
	unsupported := result.UnsupportedOperations[0]
	if unsupported.Operation.Key != "POST /v1/uploads" {
		t.Fatalf("unsupported key = %q, want %q", unsupported.Operation.Key, "POST /v1/uploads")
	}
	if len(unsupported.Reasons) != 1 || unsupported.Reasons[0] != "request body lacks application/json content" {
		t.Fatalf("unsupported reasons = %#v", unsupported.Reasons)
	}
}

func TestClassifyDriftReportsRequestBodyRefAsUnsupported(t *testing.T) {
	t.Parallel()

	baseSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)
	headSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/uploads": {
				"post": {
					"tags": ["Uploads"],
					"operationId": "CreateUpload",
					"summary": "Upload a file",
					"requestBody": {
						"$ref": "#/components/requestBodies/CreateUpload"
					}
				}
			},
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)

	baseOps, err := apisync.LoadSpec(baseSpec)
	if err != nil {
		t.Fatalf("LoadSpec(base): %v", err)
	}
	headOps, err := apisync.LoadSpec(headSpec)
	if err != nil {
		t.Fatalf("LoadSpec(head): %v", err)
	}

	result := apisync.ClassifyDrift(baseOps, headOps)
	if len(result.SupportedAdditions) != 0 {
		t.Fatalf("SupportedAdditions = %#v, want request-body ref routed to unsupported", result.SupportedAdditions)
	}
	if len(result.UnsupportedOperations) != 1 {
		t.Fatalf("UnsupportedOperations = %#v, want one ref operation", result.UnsupportedOperations)
	}
	reasons := result.UnsupportedOperations[0].Reasons
	if !hasReasonContaining(reasons, "request body $ref is not supported: #/components/requestBodies/CreateUpload") {
		t.Fatalf("unsupported reasons = %#v, want request-body ref reason", reasons)
	}
}

func TestUnsupportedReasonsRejectsJSONRequestBodyOnReadOperation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
		reason string
	}{
		{
			method: "GET",
			path:   "/v1/search",
			reason: "request body is not supported for GET operations",
		},
		{
			method: "DELETE",
			path:   "/v1/widgets",
			reason: "request body is not supported for DELETE operations",
		},
	} {
		t.Run(tc.method, func(t *testing.T) {
			op := apisync.Operation{
				OperationID:           "ReadOperationWithBody",
				Method:                tc.method,
				Path:                  tc.path,
				RequestBodyRequired:   true,
				RequestBodyMediaTypes: []string{"application/json"},
			}

			reasons := apisync.UnsupportedReasons(op)
			if len(reasons) != 1 || reasons[0] != tc.reason {
				t.Fatalf("UnsupportedReasons(%s with JSON body) = %#v, want [%q]", tc.method, reasons, tc.reason)
			}
		})
	}
}

func TestUnsupportedReasonsRejectsGeneratedParameterNames(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		want string
	}{
		{
			name: "fields[]",
			want: `unsupported parameter name "fields[]"`,
		},
		{
			name: "3ds_version",
			want: `unsupported parameter name "3ds_version"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := apisync.Operation{
				OperationID: "ListWidgets",
				Endpoint:    "widgets.list",
				Method:      "GET",
				Path:        "/v1/widgets",
				QueryParameters: []apisync.Parameter{
					{Name: tc.name, In: "query"},
				},
			}

			reasons := apisync.UnsupportedReasons(op)
			if !hasReasonContaining(reasons, tc.want) {
				t.Fatalf("UnsupportedReasons(%q) = %#v, want reason containing %q", tc.name, reasons, tc.want)
			}
		})
	}
}

func TestUnsupportedReasonsRejectsReservedGeneratedFlagNames(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"account", "json", "config", "help", "version"} {
		t.Run(name, func(t *testing.T) {
			op := apisync.Operation{
				OperationID: "ListWidgets",
				Endpoint:    "widgets.list",
				Method:      "GET",
				Path:        "/v1/widgets",
				QueryParameters: []apisync.Parameter{
					{Name: name, In: "query"},
				},
			}

			reasons := apisync.UnsupportedReasons(op)
			if !hasReasonContaining(reasons, `parameter flag name collision "`+name+`"`) {
				t.Fatalf("UnsupportedReasons(%q) = %#v, want reserved flag collision", name, reasons)
			}
		})
	}
}

func TestClassifyDriftRoutesGeneratedParameterCollisionsToUnsupported(t *testing.T) {
	t.Parallel()

	baseSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)
	headSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/collisions": {
				"get": {
					"tags": ["Collisions"],
					"operationId": "ListCollisions",
					"summary": "List collisions",
					"parameters": [
						{"name": "request-id", "in": "query"},
						{"name": "request_id", "in": "query"}
					]
				}
			},
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)

	baseOps, err := apisync.LoadSpec(baseSpec)
	if err != nil {
		t.Fatalf("LoadSpec(base): %v", err)
	}
	headOps, err := apisync.LoadSpec(headSpec)
	if err != nil {
		t.Fatalf("LoadSpec(head): %v", err)
	}

	result := apisync.ClassifyDrift(baseOps, headOps)
	if len(result.SupportedAdditions) != 0 {
		t.Fatalf("SupportedAdditions = %#v, want collision routed to unsupported", result.SupportedAdditions)
	}
	if len(result.UnsupportedOperations) != 1 {
		t.Fatalf("UnsupportedOperations = %#v, want one collision operation", result.UnsupportedOperations)
	}
	reasons := result.UnsupportedOperations[0].Reasons
	for _, want := range []string{"parameter flag name collision", "parameter variable name collision"} {
		if !hasReasonContaining(reasons, want) {
			t.Fatalf("unsupported reasons = %#v, want reason containing %q", reasons, want)
		}
	}
}

func TestParseSpecAppliesPathLevelParameters(t *testing.T) {
	t.Parallel()

	specPath := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets/{id}": {
				"parameters": [
					{"name": "id", "in": "path", "required": true, "description": "Widget id"}
				],
				"get": {
					"tags": ["Widgets"],
					"operationId": "GetWidget",
					"summary": "Get widget"
				}
			}
		}
	}`)
	ops, err := apisync.LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec(path-level params): %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("operation count = %d, want 1", len(ops))
	}
	if len(ops[0].PathParameters) != 1 || ops[0].PathParameters[0].Name != "id" {
		t.Fatalf("path parameters = %#v, want id from path item", ops[0].PathParameters)
	}
}

func TestClassifyDriftReportsPathLevelParameterChange(t *testing.T) {
	t.Parallel()

	baseSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets/{id}": {
				"parameters": [
					{"name": "id", "in": "path", "required": true, "description": "Widget id"}
				],
				"get": {
					"tags": ["Widgets"],
					"operationId": "GetWidget",
					"summary": "Get widget"
				}
			}
		}
	}`)
	headSpec := writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets/{id}": {
				"parameters": [
					{"name": "id", "in": "path", "required": true, "description": "Updated widget id"}
				],
				"get": {
					"tags": ["Widgets"],
					"operationId": "GetWidget",
					"summary": "Get widget"
				}
			}
		}
	}`)

	baseOps, err := apisync.LoadSpec(baseSpec)
	if err != nil {
		t.Fatalf("LoadSpec(base): %v", err)
	}
	headOps, err := apisync.LoadSpec(headSpec)
	if err != nil {
		t.Fatalf("LoadSpec(head): %v", err)
	}

	result := apisync.ClassifyDrift(baseOps, headOps)
	if result.NoDrift {
		t.Fatal("NoDrift = true, want path-level parameter change to be reported")
	}
	if len(result.Changes) != 1 {
		t.Fatalf("Changes = %d, want 1", len(result.Changes))
	}
	if result.Changes[0].Key != "GET /v1/widgets/{id}" {
		t.Fatalf("change key = %q, want GET /v1/widgets/{id}", result.Changes[0].Key)
	}
}

func TestGenerateEndpointFileUsesEmptyObjectForNoBodyMutation(t *testing.T) {
	t.Parallel()

	file, err := apisync.GenerateEndpointFile(apisync.Operation{
		OperationID: "ResubmitWidget",
		Endpoint:    "widgets.resubmit",
		Method:      "POST",
		Path:        "/v1/widgets/{id}/resubmit",
		PathParameters: []apisync.Parameter{
			{Name: "id", In: "path"},
		},
	}, t.TempDir())
	if err != nil {
		t.Fatalf("GenerateEndpointFile: %v", err)
	}
	got := file.Content
	for _, want := range []string{
		"body := map[string]any{}",
		"c.PostWithParamsAndHeaders(path, params, body, headers)",
		`return printGeneratedMutationOutput(cmd, flags, "POST", "widgets.resubmit", path, statusCode, data)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated content missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "var body map[string]any") {
		t.Fatalf("generated content should not declare a typed nil body for no-body mutations:\n%s", got)
	}
}

func TestGenerateEndpointFileUsesDeleteClassifier(t *testing.T) {
	t.Parallel()

	file, err := apisync.GenerateEndpointFile(apisync.Operation{
		OperationID: "DeleteWidget",
		Endpoint:    "widgets.delete",
		Method:      "DELETE",
		Path:        "/v1/widgets/{id}",
		PathParameters: []apisync.Parameter{
			{Name: "id", In: "path"},
		},
	}, t.TempDir())
	if err != nil {
		t.Fatalf("GenerateEndpointFile: %v", err)
	}
	got := file.Content
	if !strings.Contains(got, "return classifyDeleteError(err, flags)") {
		t.Fatalf("generated content missing delete classifier:\n%s", got)
	}
	if !strings.Contains(got, `return printGeneratedMutationOutput(cmd, flags, "DELETE", "widgets.delete", path, statusCode, data)`) {
		t.Fatalf("generated DELETE content missing mutation output contract:\n%s", got)
	}
	if strings.Contains(got, "return classifyAPIError(err, flags)") {
		t.Fatalf("generated DELETE content should not use generic API classifier:\n%s", got)
	}
}

func TestGenerateEndpointFileEmitsHeaderFlags(t *testing.T) {
	t.Parallel()

	file, err := apisync.GenerateEndpointFile(apisync.Operation{
		OperationID: "GetWidget",
		Endpoint:    "widgets.get",
		Method:      "GET",
		Path:        "/v1/widgets/{id}",
		PathParameters: []apisync.Parameter{
			{Name: "id", In: "path"},
		},
		HeaderParameters: []apisync.Parameter{
			{Name: "Straddle-Account-Id", In: "header"},
			{Name: "Request-Id", In: "header", Description: "Trace one request."},
		},
	}, t.TempDir())
	if err != nil {
		t.Fatalf("GenerateEndpointFile: %v", err)
	}
	got := file.Content
	for _, want := range []string{
		"var flagRequestIdHeader string",
		`headers["Request-Id"] = flagRequestIdHeader`,
		`c.GetWithHeaders(path, params, headers)`,
		`cmd.Flags().StringVar(&flagRequestIdHeader, "request-id", "", "Trace one request.")`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated content missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Straddle-Account-Id") {
		t.Fatalf("generated content should not expose Straddle-Account-Id as a header flag:\n%s", got)
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func writeSpec(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

func hasReasonContaining(reasons []string, want string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, want) {
			return true
		}
	}
	return false
}
