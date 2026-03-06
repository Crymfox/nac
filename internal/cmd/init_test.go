package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Run init with a simulated "no" to the GitHub Actions question
	// We can't easily simulate stdin in tests, so test the template rendering directly
	err := renderTemplate("templates/nac.yaml.tmpl", filepath.Join(dir, "nac.yaml"), initData{N8NVersion: "2.3.4"})
	if err != nil {
		t.Fatalf("renderTemplate(nac.yaml) failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(dir, "nac.yaml"))
	if err != nil {
		t.Fatalf("reading nac.yaml: %v", err)
	}

	// Check that the version was templated correctly
	if !contains(string(content), `n8n_version: "2.3.4"`) {
		t.Error("nac.yaml should contain the pinned version")
	}
	if !contains(string(content), "environments:") {
		t.Error("nac.yaml should contain environments section")
	}
	if !contains(string(content), "credential_types:") {
		t.Error("nac.yaml should contain credential_types section")
	}
}

func TestRenderDockerCompose(t *testing.T) {
	dir := t.TempDir()

	err := renderTemplate("templates/docker-compose.yaml.tmpl", filepath.Join(dir, "docker-compose.yaml"), initData{N8NVersion: "2.3.4"})
	if err != nil {
		t.Fatalf("renderTemplate(docker-compose) failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !contains(s, "n8nio/n8n:2.3.4") {
		t.Error("docker-compose.yaml should contain the pinned n8n image tag")
	}
	if !contains(s, "postgres:15") {
		t.Error("docker-compose.yaml should contain postgres")
	}
	if !contains(s, "redis:7-alpine") {
		t.Error("docker-compose.yaml should contain redis")
	}
	if !contains(s, "n8n-primary") {
		t.Error("docker-compose.yaml should contain n8n-primary service")
	}
	if !contains(s, "n8n-worker") {
		t.Error("docker-compose.yaml should contain n8n-worker service")
	}
}

func TestRenderGitHubActions(t *testing.T) {
	dir := t.TempDir()

	err := renderTemplate("templates/github-actions.yaml.tmpl", filepath.Join(dir, "deploy.yml"), initData{N8NVersion: "2.3.4"})
	if err != nil {
		t.Fatalf("renderTemplate(github-actions) failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "deploy.yml"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !contains(s, "Deploy n8n") {
		t.Error("GitHub Actions workflow should have the correct name")
	}
	if !contains(s, "workflow.json") {
		t.Error("should trigger on workflow.json changes")
	}
	if !contains(s, "credential.json") {
		t.Error("should trigger on credential.json changes")
	}
	if !contains(s, "nac import workflows") {
		t.Error("should run nac import workflows")
	}
	if !contains(s, "nac import credentials") {
		t.Error("should run nac import credentials")
	}
}

func TestRenderEnvTemplates(t *testing.T) {
	dir := t.TempDir()

	err := renderTemplate("templates/env.local.example.tmpl", filepath.Join(dir, ".env.local.example"), initData{})
	if err != nil {
		t.Fatalf("renderTemplate(env.local) failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".env.local.example"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !contains(s, "N8N_ENCRYPTION_KEY") {
		t.Error("should contain N8N_ENCRYPTION_KEY")
	}
	if !contains(s, "POSTGRES_USER") {
		t.Error("should contain POSTGRES_USER")
	}
}

func TestAppendGitignore_NewFile(t *testing.T) {
	dir := t.TempDir()

	err := appendGitignore(dir)
	if err != nil {
		t.Fatalf("appendGitignore() failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !contains(s, "# === nac (n8n As Code) ===") {
		t.Error("should contain nac header")
	}
	if !contains(s, ".env.local") {
		t.Error("should ignore .env.local")
	}
}

func TestAppendGitignore_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	existing := "node_modules/\n.DS_Store\n"
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644)

	err := appendGitignore(dir)
	if err != nil {
		t.Fatalf("appendGitignore() failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !contains(s, "node_modules/") {
		t.Error("should preserve existing content")
	}
	if !contains(s, "# === nac (n8n As Code) ===") {
		t.Error("should append nac section")
	}
}

func TestAppendGitignore_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Run twice
	appendGitignore(dir)
	appendGitignore(dir)

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	// Count occurrences of the header
	count := 0
	for i := 0; i < len(s); i++ {
		idx := indexOf(s[i:], "# === nac (n8n As Code) ===")
		if idx < 0 {
			break
		}
		count++
		i += idx + 1
	}
	if count != 1 {
		t.Errorf("nac section should appear exactly once, found %d times", count)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
