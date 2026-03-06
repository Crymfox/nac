package cmd

import (
	"fmt"

	"github.com/crymfox/nac/internal/config"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print nac version and pinned n8n version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("nac %s\n", config.Version)
			fmt.Printf("  commit:      %s\n", config.Commit)
			fmt.Printf("  built:       %s\n", config.Date)
			fmt.Printf("  n8n version: %s (pinned)\n", config.PinnedN8NVersion)
		},
	}
}
