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
		Short: "Manage CLAUDE.md files across large, multi-service codebases",
	}

	root.AddCommand(newInstallCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newInstallCmd() *cobra.Command {
	var update bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Resolve and install CLAUDE.md files from the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(update, &resolver.GitHubResolver{})
		},
	}

	cmd.Flags().BoolVar(&update, "update", false, "Reconcile drift between marker files and lock file")
	return cmd
}
