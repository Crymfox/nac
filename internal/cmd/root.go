package cmd

import (
	"fmt"
	"os"

	"github.com/crymfox/nac/internal/config"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	envName string
	verbose bool
	dryRun  bool
	Cfg     *config.Config
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nac",
		Short: "n8n As Code - manage n8n workflows and credentials as version-controlled files",
		Long: `nac is a CLI tool that exports n8n workflows and credentials from a Postgres
database into per-item JSON files, and imports them back into any target
environment. It enables GitOps for n8n: local development, Git versioning,
and CI/CD-driven promotion.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load .env.local file if it exists. This makes running nac locally easier.
			_ = godotenv.Load(".env.local")

			// Decide if we need to load the main nac.yaml config
			if isConfigFreeCommand(cmd) {
				return nil
			}
			return loadConfig()
		},
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "nac.yaml", "config file path")
	cmd.PersistentFlags().StringVar(&envName, "env", "local", "target environment (local/dev/staging/production)")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would change without modifying anything")

	cmd.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newExportCmd(),
		newImportCmd(),
		newUpCmd(),
		newDownCmd(),
		newLogsCmd(),
		newApiCmd(),
		newCiCmd(),
	)
	return cmd
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

// isConfigFreeCommand returns true if the command does not require a nac.yaml file.
func isConfigFreeCommand(cmd *cobra.Command) bool {
	// Check the command name itself
	name := cmd.Name()
	if name == "init" || name == "version" || name == "help" || name == "completion" || name == "" {
		return true
	}

	// Check if any parent is 'completion' or 'help' (for subcommands like 'completion bash')
	curr := cmd
	for curr != nil {
		if curr.Name() == "completion" || curr.Name() == "help" {
			return true
		}
		curr = curr.Parent()
	}

	return false
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
