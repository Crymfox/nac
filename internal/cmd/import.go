package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import",
		Aliases: []string{"sync"},
		Short:   "Import workflows or credentials from files into n8n database",
	}

	cmd.AddCommand(
		newImportWorkflowsCmd(),
		newImportCredentialsCmd(),
	)

	return cmd
}

func newImportWorkflowsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workflows",
		Short: "Import workflows from per-folder JSON files into the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] import workflows: not yet implemented (Phase 3)")
			return nil
		},
	}
}

func newImportCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials",
		Short: "Import credentials from per-folder JSON files into the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] import credentials: not yet implemented (Phase 4)")
			return nil
		},
	}
}
