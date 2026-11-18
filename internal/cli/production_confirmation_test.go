// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type confirmationTransport func(*http.Request) (*http.Response, error)

func (f confirmationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockConfirmationTransport(t *testing.T, handler confirmationTransport) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = handler
	t.Cleanup(func() { http.DefaultTransport = original })
}

func confirmationResponse(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"consented-write"}`)),
		Request:    req,
	}
}

func isolateProductionConfirmation(t *testing.T) {
	t.Helper()
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_ENVIRONMENT", "sandbox")
	t.Setenv("STRADDLE_BASE_URL", "https://production.straddle.com")
	t.Setenv("STRADDLE_API_KEY", "test_key")
}

func TestProductionWritesRequireDeliberateConsent(t *testing.T) {
	for _, method := range []string{"post", "put", "patch", "delete"} {
		for _, consent := range []struct {
			name    string
			flags   []string
			allowed bool
		}{
			{name: "agent alone", flags: []string{"--agent"}},
			{name: "noninteractive alone", flags: []string{"--no-input"}},
			{name: "piped yes is not consent"},
			{name: "agent explicit refusal", flags: []string{"--agent", "--yes=false"}},
			{name: "script explicit consent", flags: []string{"--no-input", "--yes"}, allowed: true},
			{name: "agent explicit consent", flags: []string{"--agent", "--yes"}, allowed: true},
			{name: "saved profile is not consent", flags: []string{"--profile", "legacy"}},
		} {
			t.Run(method+"/"+consent.name, func(t *testing.T) {
				isolateProductionConfirmation(t)
				if err := saveProfileStore(&profileStore{Profiles: map[string]Profile{
					"legacy": {Name: "legacy", Values: map[string]string{"yes": "true", "agent": "true"}},
				}}); err != nil {
					t.Fatal(err)
				}
				requests := 0
				mockConfirmationTransport(t, func(req *http.Request) (*http.Response, error) {
					requests++
					if req.Header.Get("Idempotency-Key") == "" {
						t.Error("consented write has no Idempotency-Key")
					}
					return confirmationResponse(req), nil
				})
				args := append([]string{"api", method, "/v1/example", "--no-cache"}, consent.flags...)
				stdout, _, err := runRootForAPITest(t, args, "yes\n")
				if consent.allowed {
					if err != nil || requests != 1 || !strings.Contains(stdout, "consented-write") {
						t.Fatalf("explicit consent: requests=%d output=%s error=%v", requests, stdout, err)
					}
					return
				}
				if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "production write confirmation required") {
					t.Fatalf("error=%v, want confirmation required with exit 2", err)
				}
				if requests != 0 || stdout != "" {
					t.Fatalf("unconfirmed write: requests=%d output=%s", requests, stdout)
				}
			})
		}
	}
}

func TestProductionConsentCoversEveryWritePath(t *testing.T) {
	for _, path := range []string{"generated", "hand-authored", "multipart", "import"} {
		t.Run(path, func(t *testing.T) {
			isolateProductionConfirmation(t)
			var args []string
			switch path {
			case "generated":
				args = []string{"customers", "delete", uploadTestID}
			case "hand-authored":
				args = []string{"bridge", "create-speedchex", "--customer-id", uploadTestID, "--speedchex-token", "test"}
			case "multipart":
				file := writeUpload(t, "proof.pdf", uploadProof)
				args = []string{"charges", "upload-authorization-proof", uploadTestID, "--file", file}
			case "import":
				file := filepath.Join(t.TempDir(), "records.jsonl")
				if err := os.WriteFile(file, []byte("{}\n{}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = []string{"import", "v1/customers", "--input", file}
			}
			requests := 0
			mockConfirmationTransport(t, func(req *http.Request) (*http.Response, error) {
				requests++
				return confirmationResponse(req), nil
			})
			args = append(args, "--agent")
			_, _, err := runRootForAPITest(t, args, "")
			if err == nil || ExitCode(err) != 2 || !strings.Contains(err.Error(), "production write confirmation required") || requests != 0 {
				t.Fatalf("unconfirmed %s: requests=%d error=%v", path, requests, err)
			}
			_, _, err = runRootForAPITest(t, append(args, "--yes"), "")
			wantRequests := 1
			if path == "import" {
				wantRequests = 2
			}
			if err != nil || requests != wantRequests {
				t.Fatalf("consented %s: requests=%d error=%v", path, requests, err)
			}
		})
	}
}

func TestProductionWriteRefusesRedirectedInput(t *testing.T) {
	isolateProductionConfirmation(t)
	requests := 0
	mockConfirmationTransport(t, func(req *http.Request) (*http.Response, error) {
		requests++
		return confirmationResponse(req), nil
	})
	file := filepath.Join(t.TempDir(), "consent.txt")
	if err := os.WriteFile(file, []byte("yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{os.DevNull, file} {
		input, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cmd := RootCmd()
		cmd.SetIn(input)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"api", "post", "/v1/example"})
		err = cmd.Execute()
		_ = input.Close()
		if err == nil || !strings.Contains(err.Error(), "stdin is not a terminal") || requests != 0 {
			t.Fatalf("redirected input %s: requests=%d error=%v", path, requests, err)
		}
	}
}

func TestProductionConsentUsesResolvedTargetAndLiveWriteIntent(t *testing.T) {
	for _, tc := range []struct {
		name, target, method string
		flags                []string
		verify               bool
		wantRequests         int
		wantRefusal          bool
	}{
		{name: "production override despite sandbox environment", target: "https://production.straddle.com", method: "post", wantRefusal: true},
		{name: "unknown remote override", target: "https://api.example.com", method: "post", wantRefusal: true},
		{name: "sandbox suffix spoof", target: "https://sandbox.straddle.com.example.com", method: "post", wantRefusal: true},
		{name: "sandbox", target: "https://sandbox.straddle.com", method: "post", wantRequests: 1},
		{name: "loopback mock", target: "http://127.0.0.1:1234", method: "post", wantRequests: 1},
		{name: "production read", target: "https://production.straddle.com", method: "get", wantRequests: 1},
		{name: "production preview", target: "https://production.straddle.com", method: "post", flags: []string{"--dry-run"}},
		{name: "verification noop", target: "https://production.straddle.com", method: "post", verify: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateProductionConfirmation(t)
			t.Setenv("STRADDLE_BASE_URL", tc.target)
			if tc.verify {
				t.Setenv("STRADDLE_VERIFY", "1")
			}
			requests := 0
			mockConfirmationTransport(t, func(req *http.Request) (*http.Response, error) {
				requests++
				return confirmationResponse(req), nil
			})
			args := append([]string{"api", tc.method, "/v1/example", "--agent", "--no-cache"}, tc.flags...)
			_, _, err := runRootForAPITest(t, args, "")
			if (err != nil) != tc.wantRefusal || requests != tc.wantRequests {
				t.Fatalf("requests=%d error=%v, want requests=%d refusal=%t", requests, err, tc.wantRequests, tc.wantRefusal)
			}
		})
	}
}

func TestRedirectCannotSupplyProductionConsent(t *testing.T) {
	isolateProductionConfirmation(t)
	t.Setenv("STRADDLE_BASE_URL", "https://sandbox.straddle.com")
	productionRequests := 0
	mockConfirmationTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "sandbox.straddle.com" {
			resp := confirmationResponse(req)
			resp.StatusCode = http.StatusTemporaryRedirect
			resp.Header.Set("Location", "https://production.straddle.com/v1/example")
			return resp, nil
		}
		productionRequests++
		return confirmationResponse(req), nil
	})
	_, _, err := runRootForAPITest(t, []string{"api", "post", "/v1/example", "--agent"}, "")
	if err == nil || !strings.Contains(err.Error(), "production write confirmation required") || productionRequests != 0 {
		t.Fatalf("redirect without consent: requests=%d error=%v", productionRequests, err)
	}
	_, _, err = runRootForAPITest(t, []string{"api", "post", "/v1/example", "--agent", "--yes"}, "")
	if err != nil || productionRequests != 1 {
		t.Fatalf("redirect with consent: requests=%d error=%v", productionRequests, err)
	}
}

type confirmationTerminalOutput struct {
	buffer bytes.Buffer
	prompt chan struct{}
	seen   bool
}

func (w *confirmationTerminalOutput) Write(p []byte) (int, error) {
	n, err := w.buffer.Write(p)
	if !w.seen && bytes.Contains(w.buffer.Bytes(), []byte("Type yes")) {
		w.seen = true
		close(w.prompt)
	}
	return n, err
}

// script supplies a real terminal to the child, proving that a human is
// prompted while pipes and character devices such as /dev/null are refused.
func TestProductionConsentAtTerminal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("terminal integration requires the Unix script utility")
	}
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("terminal integration requires script")
	}
	for _, tc := range []struct {
		name, reply, flags string
		wantWrite          bool
		wantPrompt         bool
	}{
		{name: "person confirms", reply: "yes\n", wantWrite: true, wantPrompt: true},
		{name: "person declines", reply: "no\n", wantPrompt: true},
		{name: "blank defaults to refusal", reply: "\n", wantPrompt: true},
		{name: "abbreviated response is refused", reply: "y\n", wantPrompt: true},
		{name: "end of input is refused", reply: "\x04", wantPrompt: true},
		{name: "agent at terminal still needs consent", reply: "yes\n", flags: "agent"},
		{name: "explicit interactive agent", reply: "yes\n", flags: "interactive-agent", wantWrite: true, wantPrompt: true},
		{name: "retry needs one confirmation", reply: "yes\n", flags: "retry", wantWrite: true, wantPrompt: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateProductionConfirmation(t)
			args := []string{"-q", "/dev/null", os.Args[0], "-test.run=^TestProductionConsentTerminalHelper$"}
			if runtime.GOOS == "linux" {
				quotedBinary := "'" + strings.ReplaceAll(os.Args[0], "'", "'\\''") + "'"
				args = []string{"-q", "-e", "-c", quotedBinary + " -test.run=^TestProductionConsentTerminalHelper$", "/dev/null"}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, script, args...)
			cmd.Env = append(os.Environ(), "STRADDLE_CONFIRMATION_TEST_CHILD=1", "STRADDLE_CONFIRMATION_TEST_FLAGS="+tc.flags)
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			capture := &confirmationTerminalOutput{prompt: make(chan struct{})}
			cmd.Stdout, cmd.Stderr = capture, capture
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			if tc.wantPrompt {
				select {
				case <-capture.prompt:
					if _, err := io.WriteString(input, tc.reply); err != nil {
						t.Fatal(err)
					}
				case err := <-done:
					t.Fatalf("child exited before prompting: %v\n%s", err, capture.buffer.String())
				}
			}
			if err := <-done; err != nil {
				t.Fatalf("terminal child: %v\n%s", err, capture.buffer.String())
			}
			output := capture.buffer.Bytes()
			if prompted := bytes.Contains(output, []byte("Type yes")); prompted != tc.wantPrompt {
				t.Fatalf("prompted=%t, want %t\n%s", prompted, tc.wantPrompt, output)
			}
			if prompts := bytes.Count(output, []byte("Type yes")); prompts > 1 {
				t.Fatalf("asked %d times for one invocation\n%s", prompts, output)
			}
			if wrote := bytes.Contains(output, []byte("consented-write")); wrote != tc.wantWrite {
				t.Fatalf("wrote=%t, want %t\n%s", wrote, tc.wantWrite, output)
			}
		})
	}
}

func TestProductionConsentTerminalHelper(t *testing.T) {
	if os.Getenv("STRADDLE_CONFIRMATION_TEST_CHILD") != "1" {
		return
	}
	requests := 0
	var key string
	mockConfirmationTransport(t, func(req *http.Request) (*http.Response, error) {
		requests++
		if key == "" {
			key = req.Header.Get("Idempotency-Key")
		}
		if req.Header.Get("Idempotency-Key") != key || key == "" {
			t.Error("retry changed or omitted the write's Idempotency-Key")
		}
		if os.Getenv("STRADDLE_CONFIRMATION_TEST_FLAGS") == "retry" && requests == 1 {
			return nil, errors.New("temporary transport failure")
		}
		return confirmationResponse(req), nil
	})
	cmd := RootCmd()
	args := []string{"api", "post", "/v1/example"}
	switch os.Getenv("STRADDLE_CONFIRMATION_TEST_FLAGS") {
	case "agent":
		args = append(args, "--agent")
	case "interactive-agent":
		args = append(args, "--agent", "--no-input=false")
	}
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		if ExitCode(err) != 2 {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stderr, "confirmation refused")
	}
}
