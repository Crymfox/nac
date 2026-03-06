package cmd

import (
	"fmt"
	"os"

	"github.com/crymfox/nac/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	envName string
	verbose bool
	dryRun  bool

	// Cfg holds the loaded config, available to all subcommands.
	Cfg *config.Config
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "nac",
		Short: "n8n As Code - manage n8n workflows and credentials as version-controlled files",
		Long: `nac is a CLI tool that exports n8n workflows and credentials from a Postgres
database into per-item JSON files, and imports them back into any target
environment. It enables GitOps for n8n: local development, Git versioning,
and CI/CD-driven promotion.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip config loading for init and version commands
			if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "help" {
				return nil
			}
			// Also skip for the root command itself (no subcommand)
			if cmd == cmd.Root() {
				return nil
			}
			return loadConfig()
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "nac.yaml", "config file path")
	root.PersistentFlags().StringVar(&envName, "env", "local", "target environment (local/dev/staging/production)")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would change without modifying anything")

	root.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newExportCmd(),
		newImportCmd(),
		newUpCmd(),
		newDownCmd(),
		newLogsCmd(),
	)

	return root
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GetEnvName returns the selected environment name.
func GetEnvName() string {
	return envName
}

// GetEnvironment returns the config for the selected environment.
func GetEnvironment() (*config.Environment, error) {
	if Cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	env, ok := Cfg.Environments[envName]
	if !ok {
		return nil, fmt.Errorf("environment %q not found in config", envName)
	}
	return &env, nil
}

// IsVerbose returns whether verbose mode is on.
func IsVerbose() bool {
	return verbose
}

// IsDryRun returns whether dry-run mode is on.
func IsDryRun() bool {
	return dryRun
}

func loadConfig() error {
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s (run 'nac init' to create one)", cfgFile)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	if err := config.Validate(cfg); err != nil {
		return err
	}

	Cfg = cfg
	return nil
}
