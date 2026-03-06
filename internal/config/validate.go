package config

import (
	"fmt"
	"strings"
)

// ValidationError collects multiple validation issues.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

// Add appends an error message.
func (e *ValidationError) Add(msg string) {
	e.Errors = append(e.Errors, msg)
}

// Addf appends a formatted error message.
func (e *ValidationError) Addf(format string, args ...any) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

// HasErrors returns true if any validation errors were collected.
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validate checks the config for structural correctness.
// It does not resolve env vars or test connectivity.
func Validate(cfg *Config) error {
	ve := &ValidationError{}

	if cfg.N8NVersion == "" {
		ve.Add("n8n_version is required")
	}

	// Environments
	if len(cfg.Environments) == 0 {
		ve.Add("at least one environment must be defined")
	}
	for name, env := range cfg.Environments {
		validateEnvironment(ve, name, env)
	}

	// Export
	if cfg.Export.WorkflowsDir == "" {
		ve.Add("export.workflows_dir is required")
	}
	if cfg.Export.CredentialsDir == "" {
		ve.Add("export.credentials_dir is required")
	}

	// Docker
	if cfg.Docker.ComposeFile == "" {
		ve.Add("docker.compose_file is required")
	}

	// Credential types
	for typeName, ct := range cfg.CredentialTypes {
		validateCredentialType(ve, typeName, ct)
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateEnvironment(ve *ValidationError, name string, env Environment) {
	prefix := fmt.Sprintf("environments.%s", name)

	// DB: must have either literal or env reference for host, database, user
	db := env.DB
	if db.Host == "" && db.HostEnv == "" {
		ve.Addf("%s.db: host or host_env is required", prefix)
	}
	if db.Port == 0 && db.PortEnv == "" {
		ve.Addf("%s.db: port or port_env is required", prefix)
	}
	if db.Database == "" && db.DatabaseEnv == "" {
		ve.Addf("%s.db: database or database_env is required", prefix)
	}
	if db.User == "" && db.UserEnv == "" {
		ve.Addf("%s.db: user or user_env is required", prefix)
	}
	if db.Password == "" && db.PasswordEnv == "" {
		ve.Addf("%s.db: password or password_env is required", prefix)
	}

	// Encryption key
	if env.EncryptionKeyEnv == "" {
		ve.Addf("%s.encryption_key_env is required", prefix)
	}
}

func validateCredentialType(ve *ValidationError, typeName string, ct CredentialType) {
	prefix := fmt.Sprintf("credential_types.%s", typeName)

	if len(ct.Fields) == 0 {
		ve.Addf("%s: must have at least one field", prefix)
	}
	for i, f := range ct.Fields {
		if f.Name == "" {
			ve.Addf("%s.fields[%d]: name is required", prefix, i)
		}
	}

	// OAuth2 validation
	if ct.OAuth2 != nil {
		if ct.OAuth2.TokenURL == "" {
			ve.Addf("%s.oauth2.token_url is required when oauth2 is configured", prefix)
		}
	}
}
