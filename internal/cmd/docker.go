package cmd

import (
	"github.com/crymfox/nac/internal/docker"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start the local n8n Docker Compose stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFile := Cfg.Docker.ComposeFile
			return docker.ComposeUp(composeFile)
		},
	}
}

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop the local n8n Docker Compose stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFile := Cfg.Docker.ComposeFile
			return docker.ComposeDown(composeFile)
		},
	}
}

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs [service]",
		Short: "Tail Docker Compose logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service := ""
			if len(args) > 0 {
				service = args[0]
			}
			composeFile := Cfg.Docker.ComposeFile
			return docker.ComposeLogs(composeFile, service)
		},
	}
}
