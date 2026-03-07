package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/crymfox/nac/internal/n8napi"
	"github.com/spf13/cobra"
)

func getAPIClient() *n8napi.Client {
	var apiKey, baseURL string

	// Try to get from environment configuration if config is loaded
	if Cfg != nil && envName != "" {
		if env, err := GetEnvironment(); err == nil {
			// Try to get API key from configured environment variable
			if env.APIKeyEnv != "" {
				apiKey = os.Getenv(env.APIKeyEnv)
			}

			// Try to get API URL from configured environment variable
			if env.APIUrlEnv != "" {
				baseURL = os.Getenv(env.APIUrlEnv)
			}
		}
	}

	// Fall back to default environment variables if not configured in nac.yaml
	if apiKey == "" {
		apiKey = os.Getenv("N8N_API_KEY")
	}
	if baseURL == "" {
		baseURL = os.Getenv("N8N_API_URL")
	}

	return n8napi.NewClient(baseURL, apiKey)
}

func newApiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Troubleshoot n8n via the REST API",
	}

	cmd.AddCommand(
		newApiListWorkflowsCmd(),
		newApiListExecutionsCmd(),
		newApiGetExecutionCmd(),
	)

	return cmd
}

func newApiListWorkflowsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-workflows",
		Short: "List all workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			wfs, err := client.ListWorkflows()
			if err != nil {
				return err
			}

			fmt.Printf("%-20s | %-40s | %s\n", "ID", "NAME", "ACTIVE")
			fmt.Println("--------------------------------------------------------------------------------")
			for _, wf := range wfs {
				fmt.Printf("%-20s | %-40s | %t\n", wf.ID, truncate(wf.Name, 40), wf.Active)
			}
			return nil
		},
	}
}

func newApiListExecutionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-executions [workflow_id]",
		Short: "List executions (optionally filtered by workflow ID)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			wfID := ""
			if len(args) > 0 {
				wfID = args[0]
			}

			limit, _ := cmd.Flags().GetString("limit")

			execs, err := client.ListExecutions(wfID, limit)
			if err != nil {
				return err
			}

			fmt.Printf("%-10s | %-20s | %-10s | %-10s | %-25s | %-25s\n", "ID", "WORKFLOW", "STATUS", "MODE", "STARTED", "STOPPED")
			fmt.Println("-------------------------------------------------------------------------------------------------------")
			for _, ex := range execs {
				stopped := "running"
				if ex.StoppedAt != nil {
					stopped = *ex.StoppedAt
				}
				fmt.Printf("%-10s | %-20s | %-10s | %-10s | %-25s | %-25s\n", ex.ID, ex.WorkflowID, ex.Status, ex.Mode, ex.StartedAt, stopped)
			}
			return nil
		},
	}
	cmd.Flags().String("limit", "50", "Number of executions to return")
	return cmd
}

func newApiGetExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-execution <id>",
		Short: "Get full execution details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			exec, err := client.GetExecution(args[0])
			if err != nil {
				return err
			}

			b, _ := json.MarshalIndent(exec, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
