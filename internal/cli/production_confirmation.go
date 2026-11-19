// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

type productionConfirmationError struct {
	reason string
}

func (e *productionConfirmationError) Error() string {
	return "production write confirmation required: " + e.reason + "; pass --yes to consent explicitly or --dry-run to preview"
}

// The resolved destination is authoritative: an environment variable can
// still say sandbox when --config or STRADDLE_BASE_URL targets production.
// Unknown remote hosts require consent too, so overrides cannot bypass it.
func requiresProductionConsent(target *url.URL) bool {
	host := strings.TrimSuffix(strings.ToLower(target.Hostname()), ".")
	if host == "sandbox.straddle.com" || host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return false
	}
	return true
}

func productionWriteConfirmation(cmd *cobra.Command, flags *rootFlags) func(string, string) error {
	// Consent covers this invocation's writes to one origin, including all
	// import records and retries. A redirect to another origin needs consent.
	decisions := make(map[string]error)
	return func(method, targetURL string) error {
		target, err := url.Parse(targetURL)
		if err != nil {
			return fmt.Errorf("parsing write target: %w", err)
		}
		if !requiresProductionConsent(target) || flags.yes {
			return nil
		}
		origin := target.Scheme + "://" + target.Host
		if decision, ok := decisions[origin]; ok {
			return decision
		}
		decision := confirmProductionWrite(cmd, flags, method, target, origin)
		decisions[origin] = decision
		return decision
	}
}

func confirmProductionWrite(cmd *cobra.Command, flags *rootFlags, method string, target *url.URL, origin string) error {
	if flags.noInput {
		return &productionConfirmationError{reason: "interactive input is disabled"}
	}
	input, ok := cmd.InOrStdin().(*os.File)
	if !ok || !isatty.IsTerminal(input.Fd()) {
		return &productionConfirmationError{reason: "stdin is not a terminal"}
	}
	// Do not display credentials, query parameters, or request bodies.
	fmt.Fprintf(cmd.ErrOrStderr(), "Production write: %s %s%s\nCommand: %s\n", method, origin, target.EscapedPath(), cmd.CommandPath())
	if flags.straddleAccountResolved != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Account: %s\n", flags.straddleAccountResolved)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Type yes to allow this command's writes to %s: ", origin)
	answer, err := bufio.NewReader(input).ReadSlice('\n')
	if err != nil || strings.TrimSpace(string(answer)) != "yes" {
		return &productionConfirmationError{reason: "confirmation was not received"}
	}
	return nil
}
