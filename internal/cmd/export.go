package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export workflows or credentials from n8n database to files",
	}

	cmd.AddCommand(
		newExportWorkflowsCmd(),
		newExportCredentialsCmd(),
	)

	return cmd
}

func newExportWorkflowsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workflows",
		Short: "Export workflows from the database to per-folder JSON files",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] export workflows: not yet implemented (Phase 3)")
			return nil
		},
	}
}

func newExportCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials",
		Short: "Export credentials from the database to per-folder JSON files",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] export credentials: not yet implemented (Phase 4)")
			return nil
		},
	}
}
