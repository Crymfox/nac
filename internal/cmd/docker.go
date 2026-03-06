package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start the local n8n Docker Compose stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] up: not yet implemented (Phase 5)")
			return nil
		},
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the local n8n Docker Compose stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] down: not yet implemented (Phase 5)")
			return nil
		},
	}
}

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs [service]",
		Short: "Tail Docker Compose logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("[nac] logs: not yet implemented (Phase 5)")
			return nil
		},
	}
}
