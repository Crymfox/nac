package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/crymfox/nac/internal/config"
	"github.com/spf13/cobra"
)

func newCiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "CI/CD related commands",
	}

	cmd.AddCommand(
		newCiGenerateCmd(),
	)

	return cmd
}

func newCiGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate GitHub Actions workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}

			ghDir := filepath.Join(dir, ".github", "workflows")
			if err := os.MkdirAll(ghDir, 0o755); err != nil {
				return fmt.Errorf("creating .github/workflows: %w", err)
			}

			dest := filepath.Join(ghDir, "deploy-n8n.yml")

			data := initData{
				N8NVersion: config.PinnedN8NVersion,
			}

			if err := renderTemplate("templates/github-actions.yaml.tmpl", dest, data); err != nil {
				return fmt.Errorf("generating GitHub Actions workflow: %w", err)
			}

			fmt.Printf("Generated CI workflow at: %s\n", dest)
			return nil
		},
	}
}
