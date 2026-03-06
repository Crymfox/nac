package cmd

import (
	"context"
	"fmt"

	"github.com/crymfox/nac/internal/config"
	"github.com/crymfox/nac/internal/credential"
	"github.com/crymfox/nac/internal/db"
	"github.com/crymfox/nac/internal/workflow"
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

			opts := workflow.ImportOptions{
				Client:        client,
				WorkflowsDir:  Cfg.Export.WorkflowsDir,
				MirrorDeletes: Cfg.Import.MirrorDeletes,
				PublishActive: Cfg.Import.PublishActive,
				DryRun:        IsDryRun(),
				Verbose:       IsVerbose(),
			}

			fmt.Printf("Importing workflows to %s environment...\n", GetEnvName())

			res, err := workflow.Import(ctx, opts)
			if err != nil {
				return err
			}

			fmt.Printf("\nImport complete:\n")
			fmt.Printf("  Imported:  %d\n", res.Imported)
			if Cfg.Import.MirrorDeletes {
				fmt.Printf("  Deleted:   %d (mirror mode)\n", res.Deleted)
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

func newImportCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials",
		Short: "Import credentials from per-folder JSON files into the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := GetEnvironment()
			if err != nil {
				return err
			}

			encKey, err := config.ResolveEncryptionKey(*env)
			if err != nil {
				return err
			}

			oldKeys := config.ResolveEncryptionKeyList(*env)

			ctx := context.Background()
			client, err := db.NewClient(ctx, env.DB)
			if err != nil {
				return err
			}
			defer client.Close()

			opts := credential.ImportOptions{
				Client:         client,
				CredentialsDir: Cfg.Export.CredentialsDir,
				Types:          Cfg.CredentialTypes,
				EncryptionKey:  encKey,
				OldKeys:        oldKeys,
				MirrorDeletes:  Cfg.Import.MirrorDeletes,
				DryRun:         IsDryRun(),
				Verbose:        IsVerbose(),
			}

			fmt.Printf("Importing credentials to %s environment...\n", GetEnvName())

			res, err := credential.Import(ctx, opts)
			if err != nil {
				return err
			}

			fmt.Printf("\nImport complete:\n")
			fmt.Printf("  Imported:  %d\n", res.Imported)
			if Cfg.Import.MirrorDeletes {
				fmt.Printf("  Deleted:   %d (mirror mode)\n", res.Deleted)
			}
			if len(oldKeys) > 0 {
				fmt.Printf("  Migrated:  %d (encryption key update)\n", res.Migrated)
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
