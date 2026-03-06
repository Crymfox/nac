package config

import (
	"testing"
)

func TestValidate_ValidDefaults(t *testing.T) {
	cfg := Defaults()
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate(Defaults()) failed: %v", err)
	}
}

func TestValidate_MissingN8NVersion(t *testing.T) {
	cfg := Defaults()
	cfg.N8NVersion = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when n8n_version is empty")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	assertContains(t, ve.Errors, "n8n_version is required")
}

func TestValidate_NoEnvironments(t *testing.T) {
	cfg := Defaults()
	cfg.Environments = nil

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when no environments defined")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "at least one environment must be defined")
}

func TestValidate_EnvironmentMissingDB(t *testing.T) {
	cfg := &Config{
		N8NVersion: "2.3.4",
		Environments: map[string]Environment{
			"broken": {
				DB:               DBConfig{}, // all fields missing
				EncryptionKeyEnv: "KEY",
			},
		},
		Export: ExportConfig{
			WorkflowsDir:   "wf",
			CredentialsDir: "cred",
		},
		Docker: DockerConfig{
			ComposeFile: "dc.yaml",
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail with missing DB config")
	}
	ve := err.(*ValidationError)
	// Should have errors for host, port, database, user, password
	if len(ve.Errors) < 5 {
		t.Errorf("expected at least 5 validation errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidate_EnvironmentEnvRefsOK(t *testing.T) {
	cfg := &Config{
		N8NVersion: "2.3.4",
		Environments: map[string]Environment{
			"dev": {
				DB: DBConfig{
					HostEnv:     "H",
					PortEnv:     "P",
					DatabaseEnv: "D",
					UserEnv:     "U",
					PasswordEnv: "PW",
				},
				EncryptionKeyEnv: "KEY",
			},
		},
		Export: ExportConfig{
			WorkflowsDir:   "wf",
			CredentialsDir: "cred",
		},
		Docker: DockerConfig{
			ComposeFile: "dc.yaml",
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}
}

func TestValidate_MissingEncryptionKey(t *testing.T) {
	cfg := Defaults()
	local := cfg.Environments["local"]
	local.EncryptionKeyEnv = ""
	cfg.Environments["local"] = local

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when encryption_key_env is empty")
	}
}

func TestValidate_MissingExportDirs(t *testing.T) {
	cfg := Defaults()
	cfg.Export.WorkflowsDir = ""
	cfg.Export.CredentialsDir = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when export dirs are empty")
	}
	ve := err.(*ValidationError)
	if len(ve.Errors) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(ve.Errors))
	}
}

func TestValidate_CredentialTypeNoFields(t *testing.T) {
	cfg := Defaults()
	cfg.CredentialTypes["broken"] = CredentialType{
		Fields: nil, // no fields
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when credential type has no fields")
	}
}

func TestValidate_CredentialTypeFieldNoName(t *testing.T) {
	cfg := Defaults()
	cfg.CredentialTypes["broken"] = CredentialType{
		Fields: []FieldDef{{Name: ""}},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when field has no name")
	}
}

func TestValidate_OAuth2MissingTokenURL(t *testing.T) {
	cfg := Defaults()
	cfg.CredentialTypes["broken_oauth"] = CredentialType{
		Fields: []FieldDef{{Name: "token"}},
		OAuth2: &OAuth2Config{
			TokenURL: "", // missing
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("should fail when oauth2.token_url is empty")
	}
}

func TestValidationError_Format(t *testing.T) {
	ve := &ValidationError{}
	ve.Add("first error")
	ve.Addf("error %d", 2)

	msg := ve.Error()
	if msg == "" {
		t.Fatal("error message should not be empty")
	}
	if !ve.HasErrors() {
		t.Fatal("HasErrors() should be true")
	}
}

func TestValidationError_NoErrors(t *testing.T) {
	ve := &ValidationError{}
	if ve.HasErrors() {
		t.Fatal("HasErrors() should be false for empty")
	}
}

func assertContains(t *testing.T, items []string, target string) {
	t.Helper()
	for _, item := range items {
		if item == target {
			return
		}
	}
	t.Errorf("expected errors to contain %q, got %v", target, items)
}
