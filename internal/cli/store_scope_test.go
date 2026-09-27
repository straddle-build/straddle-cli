// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/store"
	"github.com/straddle-build/straddle-cli/internal/straddleacct"
)

type recordedAPI struct {
	mu      sync.Mutex
	headers map[string][]string
}

func (a *recordedAPI) accountHeaders(path string) []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.headers[path]...)
}

// newListAPI serves one list page per resource path and records the
// Straddle-Account-Id header sent with every request.
func newListAPI(t *testing.T, pages map[string]string) (*httptest.Server, *recordedAPI) {
	t.Helper()
	api := &recordedAPI{headers: map[string][]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		api.headers[r.URL.Path] = append(api.headers[r.URL.Path], r.Header.Get(straddleacct.Header))
		api.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		body, ok := pages[r.URL.Path]
		if !ok {
			body = `{"data":[]}`
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, api
}

func isolateStoreScopeEnv(t *testing.T, baseURL string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("STRADDLE_CONFIG", filepath.Join(home, "config.toml"))
	t.Setenv("STRADDLE_PLATFORM_CONFIG", filepath.Join(home, "platform.toml"))
	t.Setenv("STRADDLE_API_KEY", "test_key")
	t.Setenv("STRADDLE_BASE_URL", baseURL)
	t.Setenv("STRADDLE_ENVIRONMENT", "")
	t.Setenv("STRADDLE_VERIFY", "")
	t.Setenv("STRADDLE_VERIFY_LIVE_HTTP", "")
}

func useAccount(t *testing.T, integrationType, account string) {
	t.Helper()
	if err := straddleacct.SaveContext(straddleacct.Context{IntegrationType: integrationType, CurrentAccount: account}); err != nil {
		t.Fatal(err)
	}
}

func searchIDs(t *testing.T, args ...string) []string {
	t.Helper()
	stdout, _, err := runRootForAPITest(t, append([]string{"--json", "--data-source", "local", "search"}, args...), "")
	if err != nil {
		t.Fatalf("search %v: %v", args, err)
	}
	var output struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode search output: %v\n%s", err, stdout)
	}
	ids := make([]string, 0, len(output.Results))
	for _, result := range output.Results {
		ids = append(ids, result.ID)
	}
	return ids
}

func TestMarketplaceCustomersStayInTheirActingAccount(t *testing.T) {
	server, api := newListAPI(t, map[string]string{
		"/v1/customers": `{"data":[{"id":"cus_shared","name":"captured under alpha"}]}`,
	})
	isolateStoreScopeEnv(t, server.URL)
	dbPath := filepath.Join(t.TempDir(), "data.db")

	useAccount(t, straddleacct.TypeMarketplace, "acct_a")
	if _, _, err := runRootForAPITest(t, []string{"--json", "sync", "--db", dbPath, "--resources", "customers"}, ""); err != nil {
		t.Fatalf("sync as acct_a: %v", err)
	}
	for _, header := range api.accountHeaders("/v1/customers") {
		if header != "" {
			t.Fatalf("marketplace customers sync sent %s %q; the policy forbids it", straddleacct.Header, header)
		}
	}
	if got := searchIDs(t, "alpha", "--db", dbPath); len(got) != 1 || got[0] != "cus_shared" {
		t.Fatalf("acct_a search = %v, want [cus_shared]", got)
	}

	useAccount(t, straddleacct.TypeMarketplace, "acct_b")
	if got := searchIDs(t, "alpha", "--db", dbPath); len(got) != 0 {
		t.Fatalf("acct_b search = %v, want nothing captured under acct_a", got)
	}

	useAccount(t, straddleacct.TypeMarketplace, "acct_a")
	if got := searchIDs(t, "alpha", "--db", dbPath); len(got) != 1 {
		t.Fatalf("acct_a search after switching back = %v, want [cus_shared]", got)
	}
}

// testStoreScope is the scope commands resolve in the test's current
// environment, so seeded rows land where the command under test reads.
func testStoreScope(t *testing.T) store.Scope {
	t.Helper()
	scope, err := localStoreScope(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func sqlIDs(t *testing.T, dbPath, query string) []string {
	t.Helper()
	stdout, _, err := runRootForAPITest(t, []string{"--json", "sql", query, "--db", dbPath}, "")
	if err != nil {
		t.Fatalf("sql %q: %v", query, err)
	}
	var rows []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("decode sql output: %v\n%s", err, stdout)
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func TestSyncSendsAccountHeaderOnlyWherePolicyAllows(t *testing.T) {
	server, api := newListAPI(t, map[string]string{
		"/v1/payments":  `{"data":[{"id":"pay_1","description":"saas payment"}]}`,
		"/v1/customers": `{"data":[{"id":"cus_1","name":"saas customer"}]}`,
	})
	isolateStoreScopeEnv(t, server.URL)
	dbPath := filepath.Join(t.TempDir(), "data.db")

	useAccount(t, straddleacct.TypeSaaS, "acct_a")
	if _, _, err := runRootForAPITest(t, []string{"--json", "sync", "--db", dbPath, "--resources", "payments,customers", "--concurrency", "1"}, ""); err != nil {
		t.Fatalf("saas sync: %v", err)
	}
	for _, path := range []string{"/v1/payments", "/v1/customers"} {
		headers := api.accountHeaders(path)
		if len(headers) == 0 || headers[0] != "acct_a" {
			t.Fatalf("saas %s headers = %q, want acct_a", path, headers)
		}
	}

	useAccount(t, straddleacct.TypeMarketplace, "acct_a")
	if out, _, err := runRootForAPITest(t, []string{"--json", "sync", "--db", dbPath, "--resources", "payments,customers", "--concurrency", "1"}, ""); err != nil {
		t.Fatalf("marketplace sync: %v\n%s", err, out)
	}
	if headers := api.accountHeaders("/v1/payments"); headers[len(headers)-1] != "acct_a" {
		t.Fatalf("marketplace payments header = %q, want acct_a", headers)
	}
	if headers := api.accountHeaders("/v1/customers"); headers[len(headers)-1] != "" {
		t.Fatalf("marketplace customers header = %q, want none (forbidden)", headers)
	}
}

func TestLocalStoreSeparatesEnvironments(t *testing.T) {
	sandbox, _ := newListAPI(t, map[string]string{"/v1/customers": `{"data":[{"id":"cus_same","name":"sandbox origin"}]}`})
	production, _ := newListAPI(t, map[string]string{"/v1/customers": `{"data":[{"id":"cus_same","name":"production origin"}]}`})
	isolateStoreScopeEnv(t, sandbox.URL)
	dbPath := filepath.Join(t.TempDir(), "data.db")
	useAccount(t, straddleacct.TypeMarketplace, "acct_a")

	for _, baseURL := range []string{sandbox.URL, production.URL} {
		t.Setenv("STRADDLE_BASE_URL", baseURL)
		if _, _, err := runRootForAPITest(t, []string{"--json", "sync", "--db", dbPath, "--resources", "customers"}, ""); err != nil {
			t.Fatalf("sync %s: %v", baseURL, err)
		}
	}
	for baseURL, want := range map[string]string{sandbox.URL: "sandbox", production.URL: "production"} {
		t.Setenv("STRADDLE_BASE_URL", baseURL)
		if got := searchIDs(t, want, "--db", dbPath); len(got) != 1 {
			t.Fatalf("%s search %q = %v, want its own row", baseURL, want, got)
		}
		other := map[string]string{"sandbox": "production", "production": "sandbox"}[want]
		if got := searchIDs(t, other, "--db", dbPath); len(got) != 0 {
			t.Fatalf("%s search %q = %v, want the other origin hidden", baseURL, other, got)
		}
	}
}

func TestMissingAccountContextIsItsOwnScope(t *testing.T) {
	server, api := newListAPI(t, map[string]string{"/v1/customers": `{"data":[{"id":"cus_platform","name":"platform level"}]}`})
	isolateStoreScopeEnv(t, server.URL)
	dbPath := filepath.Join(t.TempDir(), "data.db")

	useAccount(t, straddleacct.TypeSaaS, "")
	if _, _, err := runRootForAPITest(t, []string{"--json", "sync", "--db", dbPath, "--resources", "customers"}, ""); err != nil {
		t.Fatalf("sync without account: %v", err)
	}
	if headers := api.accountHeaders("/v1/customers"); headers[0] != "" {
		t.Fatalf("sync without an account sent %q", headers)
	}
	if got := searchIDs(t, "platform", "--db", dbPath); len(got) != 1 {
		t.Fatalf("no-account search = %v, want [cus_platform]", got)
	}
	if got := searchIDs(t, "platform", "--db", dbPath, "--account", "acct_a"); len(got) != 0 {
		t.Fatalf("acct_a search = %v, want the no-account rows hidden", got)
	}

	useAccount(t, straddleacct.TypeAccount, "")
	_, _, err := runRootForAPITest(t, []string{"--json", "--data-source", "local", "search", "platform", "--db", dbPath, "--account", "acct_a"}, "")
	if err == nil || ExitCode(err) != 2 {
		t.Fatalf("direct account --account error = %v (exit %d), want usage error", err, ExitCode(err))
	}
}

func TestLiveReadCacheNeverCrossesActingAccounts(t *testing.T) {
	var mu sync.Mutex
	names := []string{"alpha body", "bravo body"}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		name := names[min(requests, len(names)-1)]
		requests++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"api_request_id":"req"},"response_type":"object","data":{"id":"` + goldenUUID + `","name":"` + name + `"}}`))
	}))
	t.Cleanup(server.Close)
	isolateStoreScopeEnv(t, server.URL)

	for _, account := range []string{"acct_a", "acct_b"} {
		useAccount(t, straddleacct.TypeMarketplace, account)
		if _, _, err := runRootForAPITest(t, []string{"--json", "customers", "get", goldenUUID}, ""); err != nil {
			t.Fatalf("live get as %s: %v", account, err)
		}
	}
	if requests != 2 {
		t.Fatalf("API requests = %d, want 2 (acct_b must not replay acct_a's cached response)", requests)
	}
	for account, want := range map[string]string{"acct_a": "alpha body", "acct_b": "bravo body"} {
		useAccount(t, straddleacct.TypeMarketplace, account)
		stdout, _, err := runRootForAPITest(t, []string{"--json", "--data-source", "local", "customers", "get", goldenUUID}, "")
		if err != nil {
			t.Fatalf("local get as %s: %v", account, err)
		}
		if !strings.Contains(stdout, want) {
			t.Fatalf("%s local copy = %s, want %q", account, stdout, want)
		}
	}
}

func TestSQLReadsOnlyTheActingAccount(t *testing.T) {
	server, _ := newListAPI(t, map[string]string{"/v1/customers": `{"data":[{"id":"cus_sql","name":"sql marker"}]}`})
	isolateStoreScopeEnv(t, server.URL)
	dbPath := filepath.Join(t.TempDir(), "data.db")
	useAccount(t, straddleacct.TypeMarketplace, "acct_a")
	if _, _, err := runRootForAPITest(t, []string{"--json", "sync", "--db", dbPath, "--resources", "customers"}, ""); err != nil {
		t.Fatal(err)
	}
	if got := sqlIDs(t, dbPath, "SELECT id FROM customers"); len(got) != 1 || got[0] != "cus_sql" {
		t.Fatalf("acct_a sql = %v, want [cus_sql]", got)
	}
	useAccount(t, straddleacct.TypeMarketplace, "acct_b")
	if got := sqlIDs(t, dbPath, "SELECT id FROM customers UNION ALL SELECT id FROM resources"); len(got) != 0 {
		t.Fatalf("acct_b sql = %v, want nothing from acct_a", got)
	}
}
