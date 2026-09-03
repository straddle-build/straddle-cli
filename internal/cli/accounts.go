// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newAccountsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "accounts",
		Short:  "Accounts represent businesses using Straddle through your platform. Each account must complete automated...",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}

	cmd.AddCommand(newAccountsCapabilityRequestsCmd(flags))
	cmd.AddCommand(newAccountsOnboardCmd(flags))
	cmd.AddCommand(newAccountsSimulateCmd(flags))
	return cmd
}
