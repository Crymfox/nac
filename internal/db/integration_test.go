package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crymfox/nac/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDB_Integration(t *testing.T) {
	// Skip if docker isn't available
	if os.Getenv("DOCKER_HOST") == "" {
		// Just a simple check, testcontainers might also skip automatically or fail
		// Let's just try and skip if it fails to start
	}

	ctx := context.Background()

	// Locate the schema file
	wd, _ := os.Getwd()
	schemaFile := filepath.Join(filepath.Dir(filepath.Dir(wd)), "schema", "2.3.4.sql")

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("n8n"),
		postgres.WithUsername("n8n"),
		postgres.WithPassword("n8n"),
		postgres.WithInitScripts(schemaFile),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Skipf("Failed to start postgres container (is docker running?): %v", err)
		return
	}
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Initialize pgx pool directly for testing
	poolConfig, _ := pgxpool.ParseConfig(connStr)
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := &Client{pool: pool}
	defer client.Close()

	// Test Workflows
	t.Run("Workflows", func(t *testing.T) {
		wf := Workflow{
			ID:          "wf-1",
			Name:        "Test Workflow",
			Active:      true,
			IsArchived:  false,
			Nodes:       []byte(`[{"type": "trigger"}]`),
			Connections: []byte(`{}`),
			Settings:    []byte(`{}`),
		}

		err := client.UpsertWorkflow(ctx, wf)
		if err != nil {
			t.Fatalf("UpsertWorkflow failed: %v", err)
		}

		wfs, err := client.ListWorkflows(ctx)
		if err != nil {
			t.Fatalf("ListWorkflows failed: %v", err)
		}
		if len(wfs) != 1 {
			t.Fatalf("Expected 1 workflow, got %d", len(wfs))
		}
		if wfs[0].Name != "Test Workflow" {
			t.Errorf("Expected name 'Test Workflow', got %q", wfs[0].Name)
		}

		m, err := client.GetWorkflowNameIdMap(ctx)
		if err != nil {
			t.Fatalf("GetWorkflowNameIdMap failed: %v", err)
		}
		if m["Test Workflow"] != "wf-1" {
			t.Errorf("Expected map['Test Workflow'] == 'wf-1', got %q", m["Test Workflow"])
		}

		// Test Upsert Update
		wf.Active = false
		err = client.UpsertWorkflow(ctx, wf)
		if err != nil {
			t.Fatalf("UpsertWorkflow (update) failed: %v", err)
		}

		wfs, _ = client.ListWorkflows(ctx)
		if wfs[0].Active != false {
			t.Error("Workflow should be inactive after update")
		}

		// Delete
		affected, err := client.DeleteWorkflowsByNames(ctx, []string{"Test Workflow"})
		if err != nil {
			t.Fatalf("DeleteWorkflowsByNames failed: %v", err)
		}
		if affected != 1 {
			t.Errorf("Expected 1 row affected, got %d", affected)
		}

		wfs, _ = client.ListWorkflows(ctx)
		if len(wfs) != 0 {
			t.Error("Workflow should be deleted")
		}
	})

	// Test Credentials
	t.Run("Credentials", func(t *testing.T) {
		cred := Credential{
			ID:   "cred-1",
			Name: "My Secret",
			Type: "openAiApi",
			Data: "encrypted_base64",
		}

		err := client.UpsertCredential(ctx, cred)
		if err != nil {
			t.Fatalf("UpsertCredential failed: %v", err)
		}

		creds, err := client.ListCredentials(ctx)
		if err != nil {
			t.Fatalf("ListCredentials failed: %v", err)
		}
		if len(creds) != 1 {
			t.Fatalf("Expected 1 credential, got %d", len(creds))
		}
		if creds[0].Name != "My Secret" {
			t.Errorf("Expected name 'My Secret', got %q", creds[0].Name)
		}

		m, err := client.GetCredentialNameIdMap(ctx)
		if err != nil {
			t.Fatalf("GetCredentialNameIdMap failed: %v", err)
		}
		if m["My Secret"] != "cred-1" {
			t.Errorf("Expected map['My Secret'] == 'cred-1', got %q", m["My Secret"])
		}

		affected, err := client.DeleteCredentialsByNames(ctx, []string{"My Secret"})
		if err != nil {
			t.Fatalf("DeleteCredentialsByNames failed: %v", err)
		}
		if affected != 1 {
			t.Errorf("Expected 1 row affected, got %d", affected)
		}
	})
}

// Test ResolveDBConfig wrapper behavior directly
func TestNewClient_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, config.DBConfig{HostEnv: "MISSING_VAR"})
	if err == nil {
		t.Error("Expected error for missing env var")
	}
}
