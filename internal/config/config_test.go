package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nac.yaml")

	yaml := `
n8n_version: "2.3.4"
environments:
  local:
    db:
      host: localhost
      port: 5432
      database: n8n
      user: n8n
      password: n8n
    encryption_key_env: N8N_ENCRYPTION_KEY
export:
  workflows_dir: n8n_workflows
  credentials_dir: n8n_credentials
  ignore_fields:
    - createdAt
    - updatedAt
import:
  mirror_deletes: true
  publish_active: true
docker:
  compose_file: docker-compose.yaml
  auto_import_on_empty: true
credential_types:
  openAiApi:
    fields:
      - name: apiKey
        secret: true
        env_suffix: _API_KEY
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.N8NVersion != "2.3.4" {
		t.Errorf("N8NVersion = %q, want %q", cfg.N8NVersion, "2.3.4")
	}

	env, ok := cfg.Environments["local"]
	if !ok {
		t.Fatal("missing 'local' environment")
	}
	if env.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want %q", env.DB.Host, "localhost")
	}
	if env.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", env.DB.Port, 5432)
	}
	if env.EncryptionKeyEnv != "N8N_ENCRYPTION_KEY" {
		t.Errorf("EncryptionKeyEnv = %q, want %q", env.EncryptionKeyEnv, "N8N_ENCRYPTION_KEY")
	}

	if cfg.Export.WorkflowsDir != "n8n_workflows" {
		t.Errorf("Export.WorkflowsDir = %q, want %q", cfg.Export.WorkflowsDir, "n8n_workflows")
	}
	if len(cfg.Export.IgnoreFields) != 2 {
		t.Errorf("Export.IgnoreFields len = %d, want 2", len(cfg.Export.IgnoreFields))
	}

	if !cfg.Import.MirrorDeletes {
		t.Error("Import.MirrorDeletes should be true")
	}
	if !cfg.Import.PublishActive {
		t.Error("Import.PublishActive should be true")
	}

	ct, ok := cfg.CredentialTypes["openAiApi"]
	if !ok {
		t.Fatal("missing credential type 'openAiApi'")
	}
	if len(ct.Fields) != 1 {
		t.Fatalf("openAiApi fields len = %d, want 1", len(ct.Fields))
	}
	if ct.Fields[0].Name != "apiKey" {
		t.Errorf("field name = %q, want %q", ct.Fields[0].Name, "apiKey")
	}
	if !ct.Fields[0].Secret {
		t.Error("apiKey should be secret")
	}
	if ct.Fields[0].EnvSuffix != "_API_KEY" {
		t.Errorf("field env_suffix = %q, want %q", ct.Fields[0].EnvSuffix, "_API_KEY")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/nac.yaml")
	if err == nil {
		t.Fatal("Load() should fail for missing file")
	}
}

func TestLoad_EnvWithEnvRefs(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nac.yaml")

	yaml := `
n8n_version: "2.3.4"
environments:
  dev:
    db:
      host_env: DB_HOST
      port_env: DB_PORT
      database_env: DB_NAME
      user_env: DB_USER
      password_env: DB_PASS
      ssl: true
    encryption_key_env: N8N_KEY
export:
  workflows_dir: wf
  credentials_dir: cred
docker:
  compose_file: dc.yaml
credential_types: {}
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	env := cfg.Environments["dev"]
	if env.DB.HostEnv != "DB_HOST" {
		t.Errorf("DB.HostEnv = %q, want %q", env.DB.HostEnv, "DB_HOST")
	}
	if !env.DB.SSL {
		t.Error("DB.SSL should be true")
	}
}

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.N8NVersion != PinnedN8NVersion {
		t.Errorf("N8NVersion = %q, want %q", cfg.N8NVersion, PinnedN8NVersion)
	}

	if _, ok := cfg.Environments["local"]; !ok {
		t.Error("missing 'local' environment in defaults")
	}

	if cfg.Export.WorkflowsDir != "n8n_workflows" {
		t.Errorf("Export.WorkflowsDir = %q, want %q", cfg.Export.WorkflowsDir, "n8n_workflows")
	}

	// Check default credential types
	expectedTypes := []string{"openAiApi", "openRouterApi", "httpHeaderAuth", "httpQueryAuth", "supabaseApi", "youTubeOAuth2Api"}
	for _, name := range expectedTypes {
		if _, ok := cfg.CredentialTypes[name]; !ok {
			t.Errorf("missing default credential type %q", name)
		}
	}

	// Check httpHeaderAuth instances
	hha := cfg.CredentialTypes["httpHeaderAuth"]
	if len(hha.Instances) != 4 {
		t.Errorf("httpHeaderAuth instances count = %d, want 4", len(hha.Instances))
	}

	// Check YouTube OAuth2 config
	yt := cfg.CredentialTypes["youTubeOAuth2Api"]
	if yt.OAuth2 == nil {
		t.Fatal("youTubeOAuth2Api should have OAuth2 config")
	}
	if !yt.OAuth2.AutoRefresh {
		t.Error("YouTube OAuth2 AutoRefresh should be true")
	}
}

func TestResolveDBConfig_Literals(t *testing.T) {
	db := DBConfig{
		Host:     "myhost",
		Port:     5433,
		Database: "mydb",
		User:     "myuser",
		Password: "mypass",
	}

	host, port, database, user, password, err := ResolveDBConfig(db)
	if err != nil {
		t.Fatalf("ResolveDBConfig() failed: %v", err)
	}

	if host != "myhost" {
		t.Errorf("host = %q, want %q", host, "myhost")
	}
	if port != 5433 {
		t.Errorf("port = %d, want %d", port, 5433)
	}
	if database != "mydb" {
		t.Errorf("database = %q, want %q", database, "mydb")
	}
	if user != "myuser" {
		t.Errorf("user = %q, want %q", user, "myuser")
	}
	if password != "mypass" {
		t.Errorf("password = %q, want %q", password, "mypass")
	}
}

func TestResolveDBConfig_EnvVars(t *testing.T) {
	t.Setenv("TEST_DB_HOST", "remotehost")
	t.Setenv("TEST_DB_PORT", "25432")
	t.Setenv("TEST_DB_NAME", "remotedb")
	t.Setenv("TEST_DB_USER", "remoteuser")
	t.Setenv("TEST_DB_PASS", "remotepass")

	db := DBConfig{
		Host:        "fallback",
		HostEnv:     "TEST_DB_HOST",
		Port:        5432,
		PortEnv:     "TEST_DB_PORT",
		DatabaseEnv: "TEST_DB_NAME",
		UserEnv:     "TEST_DB_USER",
		PasswordEnv: "TEST_DB_PASS",
	}

	host, port, database, user, password, err := ResolveDBConfig(db)
	if err != nil {
		t.Fatalf("ResolveDBConfig() failed: %v", err)
	}

	// Env vars should take precedence over literals
	if host != "remotehost" {
		t.Errorf("host = %q, want %q", host, "remotehost")
	}
	if port != 25432 {
		t.Errorf("port = %d, want %d", port, 25432)
	}
	if database != "remotedb" {
		t.Errorf("database = %q, want %q", database, "remotedb")
	}
	if user != "remoteuser" {
		t.Errorf("user = %q, want %q", user, "remoteuser")
	}
	if password != "remotepass" {
		t.Errorf("password = %q, want %q", password, "remotepass")
	}
}

func TestResolveDBConfig_MissingEnvVar(t *testing.T) {
	db := DBConfig{
		HostEnv: "NONEXISTENT_HOST_VAR_12345",
	}

	_, _, _, _, _, err := ResolveDBConfig(db)
	if err == nil {
		t.Fatal("should fail when env var is not set")
	}
}

func TestResolveDBConfig_InvalidPort(t *testing.T) {
	t.Setenv("TEST_BAD_PORT", "notanumber")

	db := DBConfig{
		Host:     "h",
		PortEnv:  "TEST_BAD_PORT",
		Database: "d",
		User:     "u",
		Password: "p",
	}

	_, _, _, _, _, err := ResolveDBConfig(db)
	if err == nil {
		t.Fatal("should fail for non-numeric port")
	}
}

func TestResolveEncryptionKey(t *testing.T) {
	t.Setenv("MY_ENC_KEY", "supersecret")

	env := Environment{EncryptionKeyEnv: "MY_ENC_KEY"}
	key, err := ResolveEncryptionKey(env)
	if err != nil {
		t.Fatalf("ResolveEncryptionKey() failed: %v", err)
	}
	if key != "supersecret" {
		t.Errorf("key = %q, want %q", key, "supersecret")
	}
}

func TestResolveEncryptionKey_Missing(t *testing.T) {
	env := Environment{EncryptionKeyEnv: "NONEXISTENT_KEY_12345"}
	_, err := ResolveEncryptionKey(env)
	if err == nil {
		t.Fatal("should fail when env var is not set")
	}
}

func TestResolveEncryptionKeyList(t *testing.T) {
	t.Setenv("MY_KEY_LIST", "key1, key2 ,key3")

	env := Environment{EncryptionKeyListEnv: "MY_KEY_LIST"}
	keys := ResolveEncryptionKeyList(env)
	if len(keys) != 3 {
		t.Fatalf("keys len = %d, want 3", len(keys))
	}
	if keys[0] != "key1" || keys[1] != "key2" || keys[2] != "key3" {
		t.Errorf("keys = %v, want [key1, key2, key3]", keys)
	}
}

func TestResolveEncryptionKeyList_Empty(t *testing.T) {
	env := Environment{}
	keys := ResolveEncryptionKeyList(env)
	if keys != nil {
		t.Errorf("keys should be nil, got %v", keys)
	}
}
