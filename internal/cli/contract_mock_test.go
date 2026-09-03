// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/apisync"
	"github.com/straddle-build/straddle-cli/internal/surface"
	"sigs.k8s.io/yaml"
)

const (
	contractMockSpecPath = "../../spec.yaml"
	contractPathValue    = "550e8400-e29b-41d4-a716-446655440000"
	contractMockStartup  = 60 * time.Second
)

var contractValuesByFormat = map[string]string{
	"uuid":      contractPathValue,
	"date":      "2026-01-15",
	"date-time": "2026-01-15T10:00:00Z",
	"uri":       "https://example.com",
	"email":     "test@example.com",
	"ipv4":      "203.0.113.10",
}

var contractValuesByWireKey = map[string]string{
	"Idempotency-Key":                          "idem-0123456789",
	"/routing_number":                          "021000021",
	"/bank_account/routing_number":             "021000021",
	"/business_profile/tax_id":                 "123456789",
	"/business_profile/phone":                  "+15555550123",
	"/business_profile/support_channels/phone": "+15555550123",
	"/phone":                                "+15555550123",
	"/mobile_number":                        "+15555550123",
	"/business_profile/address/state":       "CA",
	"/address/state":                        "CA",
	"/business_profile/address/postal_code": "12345",
	"/ssn_last4":                            "1234",
}

type contractOperationKey struct {
	method string
	path   string
}

func (k contractOperationKey) String() string {
	return k.method + " " + k.path
}

type contractArguments struct {
	all      []string
	required []string
}

type contractInvocation struct {
	name    string
	cmd     *cobra.Command
	surface surface.Surface
	args    []string
}

type contractOperation struct {
	responses map[string]json.RawMessage
	secured   bool
}

type contractDocument struct {
	Security []map[string][]string                      `json:"security"`
	Paths    map[string]map[string]rawContractOperation `json:"paths"`
}

type rawContractOperation struct {
	Responses map[string]json.RawMessage `json:"responses"`
	Security  *[]map[string][]string     `json:"security"`
}

type recordedContractResponse struct {
	statusCode    int
	body          []byte
	authorization string
}

type contractRecordingTransport struct {
	base http.RoundTripper

	mu        sync.Mutex
	responses []recordedContractResponse
}

func (t *contractRecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	t.mu.Lock()
	t.responses = append(t.responses, recordedContractResponse{
		statusCode:    resp.StatusCode,
		body:          append([]byte(nil), body...),
		authorization: req.Header.Get("Authorization"),
	})
	t.mu.Unlock()
	return resp, nil
}

func (t *contractRecordingTransport) reset() {
	t.mu.Lock()
	t.responses = nil
	t.mu.Unlock()
}

func (t *contractRecordingTransport) take() []recordedContractResponse {
	t.mu.Lock()
	defer t.mu.Unlock()
	responses := append([]recordedContractResponse(nil), t.responses...)
	t.responses = nil
	return responses
}

func TestContractMockServer(t *testing.T) {
	requireContractMock(t)

	surfaces, unsupported, err := apisync.DeriveSurfaces(contractMockSpecPath)
	if err != nil {
		t.Fatalf("derive surfaces: %v", err)
	}
	surfaces = supportedContractSurfaces(surfaces, unsupported)
	arguments := contractArgumentTable(surfaces)
	operations := loadContractOperations(t, surfaces)
	invocations := make([]contractInvocation, 0, len(surfaces)*2)
	for _, s := range surfaces {
		key := contractOperationKey{method: s.Method, path: s.Path}
		sets := arguments[key]
		invocations = append(invocations,
			newDerivedContractInvocation(s, "all flags", sets.all),
			newDerivedContractInvocation(s, "required flags", sets.required),
		)
	}

	runContractInvocations(t, invocations, operations)
}

func TestContractMockServerRegisteredCommands(t *testing.T) {
	requireContractMock(t)

	registered := registeredSurfaces()
	if len(registered) == 0 {
		t.Skip("production command validation waits for phase 3 endpoint regeneration; registeredSurfaces() is empty")
	}
	surfaces, unsupported, err := apisync.DeriveSurfaces(contractMockSpecPath)
	if err != nil {
		t.Fatalf("derive surfaces: %v", err)
	}
	surfaces = supportedContractSurfaces(surfaces, unsupported)
	arguments := contractArgumentTable(surfaces)
	operations := loadContractOperations(t, registered)
	invocations := registeredContractInvocations(t, registered, arguments)

	runContractInvocations(t, invocations, operations)
}

func requireContractMock(t *testing.T) {
	t.Helper()
	if os.Getenv("STRADDLE_CONTRACT_MOCK") != "1" {
		t.Skip("set STRADDLE_CONTRACT_MOCK=1 to run Scalar contract validation")
	}
}

func supportedContractSurfaces(surfaces []surface.Surface, unsupported []apisync.UnsupportedOperation) []surface.Surface {
	unsupportedKeys := make(map[contractOperationKey]struct{}, len(unsupported))
	for _, item := range unsupported {
		unsupportedKeys[contractOperationKey{method: item.Operation.Method, path: item.Operation.Path}] = struct{}{}
	}
	supported := make([]surface.Surface, 0, len(surfaces)-len(unsupportedKeys))
	for _, s := range surfaces {
		key := contractOperationKey{method: s.Method, path: s.Path}
		if _, found := unsupportedKeys[key]; !found {
			supported = append(supported, s)
		}
	}
	return supported
}

func contractArgumentTable(surfaces []surface.Surface) map[contractOperationKey]contractArguments {
	table := make(map[contractOperationKey]contractArguments, len(surfaces))
	for _, s := range surfaces {
		all := appendContractPathArguments(nil, s)
		required := append([]string(nil), all...)
		for _, flag := range s.Flags {
			all = appendContractFlag(all, flag)
			if flag.Required {
				required = appendContractFlag(required, flag)
			}
		}
		table[contractOperationKey{method: s.Method, path: s.Path}] = contractArguments{
			all:      all,
			required: required,
		}
	}
	return table
}

func appendContractPathArguments(args []string, s surface.Surface) []string {
	for range s.PathParams {
		args = append(args, contractPathValue)
	}
	return args
}

func appendContractFlag(args []string, flag surface.Flag) []string {
	values := contractFlagValues(flag)
	for _, value := range values {
		args = append(args, "--"+flag.Name, value)
	}
	return args
}

func contractFlagValues(flag surface.Flag) []string {
	first := contractScalarValue(flag)
	if !flag.Array {
		return []string{first}
	}
	second := first
	if len(flag.Enum) > 1 {
		second = flag.Enum[1]
	}
	return []string{first, second}
}

func contractScalarValue(flag surface.Flag) string {
	if len(flag.Enum) > 0 {
		return flag.Enum[0]
	}
	if value, ok := contractValuesByWireKey[flag.Key]; ok {
		return value
	}
	if value, ok := contractValuesByFormat[flag.Format]; ok {
		return value
	}
	switch flag.Kind {
	case surface.KindString:
		return "x"
	case surface.KindInteger:
		return "1"
	case surface.KindNumber:
		return "1.5"
	case surface.KindBoolean:
		return "true"
	case surface.KindJSON:
		if flag.Key == "/metadata" || flag.Key == "/compliance_profile" {
			return "{}"
		}
		return "[]"
	default:
		panic(fmt.Sprintf("unsupported contract flag kind %q", flag.Kind))
	}
}

func newDerivedContractInvocation(s surface.Surface, name string, args []string) contractInvocation {
	flags := &rootFlags{noCache: true, timeout: 10 * time.Second}
	root := &cobra.Command{Use: "straddle", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Show request without sending")
	cmd := &cobra.Command{
		Use: "fixture",
		Annotations: map[string]string{
			"straddle:endpoint":     s.Endpoint,
			"straddle:operation-id": s.OperationID,
			"straddle:method":       s.Method,
			"straddle:path":         s.Path,
		},
	}
	bind := bindSurface(cmd, flags, s)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		req, err := bind(args)
		if err != nil {
			return err
		}
		return executeSurface(cmd, flags, s, req)
	}
	root.AddCommand(cmd)
	return contractInvocation{
		name:    name,
		cmd:     root,
		surface: s,
		args:    append([]string{"fixture"}, args...),
	}
}

func registeredContractInvocations(t *testing.T, surfaces []surface.Surface, arguments map[contractOperationKey]contractArguments) []contractInvocation {
	t.Helper()
	root := RootCmd()
	commands := annotatedContractCommands(t, root)
	invocations := make([]contractInvocation, 0, len(surfaces)*2)
	for _, s := range surfaces {
		key := contractOperationKey{method: s.Method, path: s.Path}
		sets, ok := arguments[key]
		if !ok {
			t.Fatalf("%s has no derived argument set", key)
		}
		annotated, ok := commands[key]
		if !ok {
			t.Fatalf("%s has no annotated production command", key)
		}
		prefix := contractCommandPrefix(annotated)
		invocations = append(invocations,
			contractInvocation{
				name:    "all flags",
				cmd:     RootCmd(),
				surface: s,
				args:    append(append(append([]string(nil), prefix...), sets.all...), "--no-cache"),
			},
			contractInvocation{
				name:    "required flags",
				cmd:     RootCmd(),
				surface: s,
				args:    append(append(append([]string(nil), prefix...), sets.required...), "--no-cache"),
			},
		)
	}
	return invocations
}

func annotatedContractCommands(t *testing.T, root *cobra.Command) map[contractOperationKey]*cobra.Command {
	t.Helper()
	commands := map[contractOperationKey]*cobra.Command{}
	var visit func(*cobra.Command)
	visit = func(cmd *cobra.Command) {
		method := cmd.Annotations["straddle:method"]
		path := cmd.Annotations["straddle:path"]
		if method != "" && path != "" {
			key := contractOperationKey{method: method, path: path}
			if prior, exists := commands[key]; exists {
				t.Fatalf("%s is declared by both %q and %q", key, prior.CommandPath(), cmd.CommandPath())
			}
			commands[key] = cmd
		}
		for _, child := range cmd.Commands() {
			visit(child)
		}
	}
	visit(root)
	return commands
}

func contractCommandPrefix(cmd *cobra.Command) []string {
	var reversed []string
	for current := cmd; current.Parent() != nil; current = current.Parent() {
		reversed = append(reversed, current.Name())
	}
	prefix := make([]string, len(reversed))
	for i := range reversed {
		prefix[len(reversed)-1-i] = reversed[i]
	}
	return prefix
}

func loadContractOperations(t *testing.T, surfaces []surface.Surface) map[contractOperationKey]contractOperation {
	t.Helper()
	data, err := os.ReadFile(contractMockSpecPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var document contractDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse contract: %v", err)
	}

	operations := make(map[contractOperationKey]contractOperation, len(surfaces))
	for _, s := range surfaces {
		key := contractOperationKey{method: s.Method, path: s.Path}
		raw, ok := document.Paths[s.Path][strings.ToLower(s.Method)]
		if !ok {
			t.Fatalf("%s is absent from the contract", key)
		}
		secured := len(document.Security) > 0
		if raw.Security != nil {
			secured = len(*raw.Security) > 0
		}
		operations[key] = contractOperation{responses: raw.Responses, secured: secured}
	}
	return operations
}

func runContractInvocations(t *testing.T, invocations []contractInvocation, operations map[contractOperationKey]contractOperation) {
	t.Helper()
	baseURL := startContractMockServer(t)
	isolateContractMockConfig(t, baseURL)
	recorder := installContractRecorder(t)

	var failures []string
	for _, invocation := range invocations {
		key := contractOperationKey{method: invocation.surface.Method, path: invocation.surface.Path}
		operation, ok := operations[key]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s [%s]: missing response contract", key, invocation.name))
			continue
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		invocation.cmd.SetOut(&stdout)
		invocation.cmd.SetErr(&stderr)
		invocation.cmd.SetIn(strings.NewReader(""))
		invocation.cmd.SetArgs(invocation.args)
		recorder.reset()
		err := invocation.cmd.Execute()
		responses := recorder.take()
		if failure := contractInvocationFailure(invocation, operation, err, responses, stderr.String()); failure != "" {
			failures = append(failures, failure)
		}
	}
	if len(failures) > 0 {
		t.Fatalf("contract mock validation failed:\n%s", strings.Join(failures, "\n\n"))
	}
}

func contractInvocationFailure(invocation contractInvocation, operation contractOperation, err error, responses []recordedContractResponse, stderr string) string {
	key := contractOperationKey{method: invocation.surface.Method, path: invocation.surface.Path}
	var reasons []string
	if err != nil {
		var classified *cliError
		if operation.secured && errors.As(err, &classified) && classified.code == 4 {
			reasons = append(reasons, fmt.Sprintf("security failed through CLI auth error class: %v", err))
		} else {
			reasons = append(reasons, fmt.Sprintf("CLI error with exit code %d: %v", ExitCode(err), err))
		}
	}
	if len(responses) == 0 {
		reasons = append(reasons, "the CLI sent no HTTP request")
	} else {
		for _, response := range responses {
			if response.statusCode == http.StatusNotFound {
				reasons = append(reasons, "mock returned 404")
			}
			if response.statusCode == http.StatusUnprocessableEntity {
				reasons = append(reasons, "mock returned 422\n"+formatContractProblem(response.body))
			}
		}
		last := responses[len(responses)-1]
		if !declaresContractStatus(operation.responses, last.statusCode) {
			reasons = append(reasons, fmt.Sprintf("mock returned undeclared status %d; declared responses: %s", last.statusCode, declaredContractStatuses(operation.responses)))
		}
		if operation.secured && last.authorization != "Bearer test-key" {
			reasons = append(reasons, fmt.Sprintf("Authorization header = %q, want %q", last.authorization, "Bearer test-key"))
		}
	}
	if len(reasons) == 0 {
		return ""
	}
	if strings.TrimSpace(stderr) != "" {
		reasons = append(reasons, "CLI stderr: "+strings.TrimSpace(stderr))
	}
	return fmt.Sprintf("%s [%s]: %s", key, invocation.name, strings.Join(reasons, "; "))
}

func declaresContractStatus(responses map[string]json.RawMessage, statusCode int) bool {
	status := strconv.Itoa(statusCode)
	if _, ok := responses[status]; ok {
		return true
	}
	if _, ok := responses["default"]; ok {
		return true
	}
	for declared := range responses {
		declared = strings.ToUpper(declared)
		if len(declared) == 3 && declared[1:] == "XX" && status[0] == declared[0] {
			return true
		}
	}
	return false
}

func declaredContractStatuses(responses map[string]json.RawMessage) string {
	statuses := make([]string, 0, len(responses))
	for status := range responses {
		statuses = append(statuses, status)
	}
	slices.Sort(statuses)
	return strings.Join(statuses, ", ")
}

func formatContractProblem(body []byte) string {
	var formatted bytes.Buffer
	if json.Indent(&formatted, body, "", "  ") == nil {
		return formatted.String()
	}
	return string(body)
}

func installContractRecorder(t *testing.T) *contractRecordingTransport {
	t.Helper()
	original := http.DefaultTransport
	recorder := &contractRecordingTransport{base: original}
	http.DefaultTransport = recorder
	t.Cleanup(func() {
		http.DefaultTransport = original
	})
	return recorder
}

func isolateContractMockConfig(t *testing.T, baseURL string) {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("HOME", configDir)
	t.Setenv("STRADDLE_CONFIG", filepath.Join(configDir, "config.toml"))
	t.Setenv("STRADDLE_PLATFORM_CONFIG", filepath.Join(configDir, "platform.toml"))
	t.Setenv("STRADDLE_API_KEY", "test-key")
	t.Setenv("STRADDLE_BASE_URL", baseURL)
	t.Setenv("STRADDLE_VERIFY", "")
	t.Setenv("STRADDLE_VERIFY_LIVE_HTTP", "")
}

func startContractMockServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve mock port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release mock port: %v", err)
	}

	stderr, err := os.CreateTemp(t.TempDir(), "scalar-stderr-*.log")
	if err != nil {
		t.Fatalf("create Scalar stderr log: %v", err)
	}
	t.Cleanup(func() {
		_ = stderr.Close()
	})
	cmd := exec.Command("npx", "--yes", "@scalar/cli@2.1.0", "document", "mock", contractMockSpecPath, "--port", strconv.Itoa(port))
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start Scalar mock server: %v", err)
	}
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
			return
		default:
		}
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	})

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.NewTimer(contractMockStartup)
	defer deadline.Stop()
	retry := time.NewTicker(50 * time.Millisecond)
	defer retry.Stop()
	for {
		conn, dialErr := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return "http://" + address
		}
		select {
		case <-done:
			t.Fatalf("Scalar mock server exited before accepting connections: %v\nstderr:\n%s", waitErr, readContractMockStderr(stderr.Name()))
		case <-deadline.C:
			t.Fatalf("Scalar mock server did not accept connections within %s\nstderr:\n%s", contractMockStartup, readContractMockStderr(stderr.Name()))
		case <-retry.C:
		}
	}
}

func readContractMockStderr(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("read stderr log: %v", err)
	}
	return string(data)
}
