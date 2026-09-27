package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const straddleCustomerEnvelope = `{"meta":{"api_request_id":"r1"},"response_type":"object","data":{"id":"cus_1","status":"verified","name":"A"}}`

func newCustomerServer(t *testing.T) {
	t.Helper()
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_API_KEY", "test_key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(straddleCustomerEnvelope))
	}))
	t.Cleanup(server.Close)
	t.Setenv("STRADDLE_BASE_URL", server.URL)
}

func TestSelectReadIsResourceRelative(t *testing.T) {
	newCustomerServer(t)

	stdout, _, err := runRootForAPITest(t, []string{"--agent", "--no-cache", "--select", "id,status", "customers", "get", "cus_1"}, "")
	if err != nil {
		t.Fatalf("customers get --select id,status: %v", err)
	}
	results, ok := decodeAPIEnvelope(t, stdout)["results"].(map[string]any)
	if !ok {
		t.Fatalf("results is not an object: %s", stdout)
	}
	data, ok := results["data"].(map[string]any)
	if !ok || len(data) != 2 || data["id"] != "cus_1" || data["status"] != "verified" {
		t.Fatalf("results.data = %v, want only id and status", results["data"])
	}
	if results["meta"] == nil {
		t.Fatalf("API meta dropped from results: %s", stdout)
	}
}

func TestSelectReadWithoutMatchFailsWithDiagnostic(t *testing.T) {
	// meta.source reaches the API's own meta object, so there are no resource
	// fields to suggest; the diagnostic still rejects it.
	for selector, want := range map[string]string{
		"no_such_field": "this response has: id, name, status",
		"results.id":    "this response has: id, name, status",
		"meta.source":   "matched no fields",
	} {
		t.Run(selector, func(t *testing.T) {
			newCustomerServer(t)
			stdout, _, err := runRootForAPITest(t, []string{"--agent", "--no-cache", "--select", selector, "customers", "get", "cus_1"}, "")
			if err == nil {
				t.Fatalf("want an error, got output %s", stdout)
			}
			if ExitCode(err) != 2 {
				t.Fatalf("exit code = %d, want 2 (%v)", ExitCode(err), err)
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("diagnostic = %v, want containing %q", err, want)
			}
			if stdout != "" {
				t.Fatalf("a rejected selector still printed %s", stdout)
			}
		})
	}
}

func TestSelectWithoutMatchKeepsCompletedWriteResult(t *testing.T) {
	newCustomerServer(t)

	cmd := RootCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetIn(strings.NewReader("{}"))
	cmd.SetArgs([]string{"--json", "--select", "no_such_field", "api", "post", "/v1/customers", "--stdin"})

	// No t.Parallel(): captureStderr swaps the process-global os.Stderr.
	var execErr error
	stderr := captureStderr(t, func() {
		execErr = cmd.Execute()
	})
	if execErr != nil {
		t.Fatalf("completed write returned error: %v", execErr)
	}
	if !strings.Contains(stderr, `warning: --select "no_such_field" matched no fields`) {
		t.Fatalf("stderr missing selector warning; got:\n%s", stderr)
	}
	env := decodeAPIEnvelope(t, stdout.String())
	if env["success"] != true {
		t.Fatalf("success = %v, want true: %v", env["success"], env)
	}
	body, _ := env["data"].(map[string]any)
	resource, _ := body["data"].(map[string]any)
	if resource["id"] != "cus_1" || resource["name"] != "A" {
		t.Fatalf("write response was not kept in full: %v", env["data"])
	}
}
