package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/straddleacct"
)

// A dry run is the preview agents and people check before a write, so it must
// show the scoping and idempotency headers the real request would carry, and
// mask any other header the way it masks Authorization.
func TestDryRunShowsTheHeadersTheRequestWouldSend(t *testing.T) {
	isolateAPIConfig(t)
	t.Setenv("STRADDLE_API_KEY", "test_key")
	t.Setenv("STRADDLE_BASE_URL", "http://127.0.0.1:1")
	if err := straddleacct.SaveContext(straddleacct.Context{
		IntegrationType: straddleacct.TypeMarketplace, CurrentAccount: "acct_sticky",
	}); err != nil {
		t.Fatal(err)
	}
	customHeader := "[headers]\nX-Partner-Secret = \"partner-secret-value\"\n"
	if err := os.WriteFile(os.Getenv("STRADDLE_CONFIG"), []byte(customHeader), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		args   []string
		want   []string
		absent []string
	}{
		{
			name:   "account-scoped charge shows the acting account and idempotency key",
			args:   append([]string{"charges", "create", "--idempotency-key=idem-charge"}, goldenInvocations["charges.create"]...),
			want:   []string{"  Straddle-Account-Id: acct_sticky\n", "  Idempotency-Key: idem-charge\n", "  X-Partner-Secret: ****alue\n"},
			absent: []string{"partner-secret-value"},
		},
		{
			name:   "platform-level account create sends no acting account",
			args:   append([]string{"accounts", "create", "--idempotency-key=idem-account"}, goldenInvocations["accounts.create"]...),
			want:   []string{"  Idempotency-Key: idem-account\n"},
			absent: []string{"Straddle-Account-Id", "partner-secret-value"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err, _ := capturedRun(t, append([]string{"--dry-run", "--agent"}, tc.args...))
			if err != nil {
				t.Fatalf("dry run: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(stderr, want) {
					t.Errorf("dry run missing %q:\n%s", want, stderr)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(stderr, absent) {
					t.Errorf("dry run shows %q, which the request would not send:\n%s", absent, stderr)
				}
			}
		})
	}
}
