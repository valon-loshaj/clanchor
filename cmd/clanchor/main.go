package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/valon-loshaj/clanchor/internal/resolver"
)

func main() {
	root := &cobra.Command{
		Use:   "clanchor",
		Short: "Package manager for .claude directories",
	}

	root.AddCommand(newInstallCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newRemoveCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newInstallCmd() *cobra.Command {
	var update bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Resolve and install packages and CLAUDE.md files from the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallV2(update, &resolver.GitHubResolver{})
		},
	}

	cmd.Flags().BoolVar(&update, "update", false, "Reconcile drift between manifest and lock file")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show installed packages and detect drift",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <package>",
		Short: "Remove a package from manifest and delete its files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args[0])
		},
	}
}
