package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/crymfox/nac/internal/config"
	"github.com/crymfox/nac/internal/credential"
	"github.com/crymfox/nac/internal/db"
	"github.com/crymfox/nac/internal/workflow"
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
			env, err := GetEnvironment()
			if err != nil {
				return err
			}

			ctx := context.Background()
			client, err := db.NewClient(ctx, env.DB)
			if err != nil {
				return err
			}
			defer client.Close()

			opts := workflow.ExportOptions{
				Client:       client,
				WorkflowsDir: Cfg.Export.WorkflowsDir,
				IgnoreFields: Cfg.Export.IgnoreFields,
				DryRun:       IsDryRun(),
				Verbose:      IsVerbose(),
			}

			fmt.Printf("Exporting workflows from %s environment...\n", GetEnvName())

			res, err := workflow.Export(ctx, opts)
			if err != nil {
				return err
			}

			fmt.Printf("\nExport complete:\n")
			fmt.Printf("  Updated:   %d\n", res.Updated)
			fmt.Printf("  Unchanged: %d\n", res.Unchanged)
			fmt.Printf("  Removed:   %d\n", res.Removed)

			if len(res.Errors) > 0 {
				fmt.Printf("\nWarnings:\n")
				for _, err := range res.Errors {
					fmt.Printf("  - %v\n", err)
				}
			}

			return nil
		},
	}
}

func newExportCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials",
		Short: "Export credentials from the database to per-folder JSON files",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnvironment()
			if err != nil {
				return err
			}

			encKey, err := config.ResolveEncryptionKey(*env)
			if err != nil {
				return err
			}

			ctx := context.Background()
			client, err := db.NewClient(ctx, env.DB)
			if err != nil {
				return err
			}
			defer client.Close()

			// Determine which .env file to update with decrypted secrets
			updateEnv := ""
			envName := GetEnvName()
			if envName == "local" {
				updateEnv = ".env.local"
			} else {
				// Look for .env.<env> or .env.remote.<env>
				candidates := []string{
					".env." + envName,
					".env.remote." + envName,
				}
				for _, c := range candidates {
					if _, err := os.Stat(c); err == nil {
						updateEnv = c
						break
					}
				}
			}

			opts := credential.ExportOptions{
				Client:         client,
				CredentialsDir: Cfg.Export.CredentialsDir,
				Types:          Cfg.CredentialTypes,
				EncryptionKey:  encKey,
				UpdateEnvFile:  updateEnv,
				DryRun:         IsDryRun(),
				Verbose:        IsVerbose(),
			}

			fmt.Printf("Exporting credentials from %s environment...\n", envName)

			res, err := credential.Export(ctx, opts)
			if err != nil {
				return err
			}

			fmt.Printf("\nExport complete:\n")
			fmt.Printf("  Updated:   %d\n", res.Updated)
			fmt.Printf("  Unchanged: %d\n", res.Unchanged)
			fmt.Printf("  Removed:   %d\n", res.Removed)
			if updateEnv != "" {
				fmt.Printf("  Secrets written to: %s\n", updateEnv)
			}

			if len(res.Errors) > 0 {
				fmt.Printf("\nWarnings:\n")
				for _, err := range res.Errors {
					fmt.Printf("  - %v\n", err)
				}
			}

			return nil
		},
	}
}
