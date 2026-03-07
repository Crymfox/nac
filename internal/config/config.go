package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a nac.yaml config file.
// It resolves env var references but does NOT validate the config -
// call Validate() on the result for that.
func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Defaults returns a Config with sensible defaults (matching what nac init generates).
func Defaults() *Config {
	return &Config{
		N8NVersion: PinnedN8NVersion,
		Environments: map[string]Environment{
			"local": {
				DB: DBConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "n8n",
					User:     "n8n",
					Password: "n8n",
					SSL:      false,
				},
				EncryptionKeyEnv: "N8N_ENCRYPTION_KEY",
				APIKeyEnv:        "N8N_API_KEY",
				APIUrlEnv:        "N8N_API_URL",
			},
		},
		Export: ExportConfig{
			WorkflowsDir:   "n8n_workflows",
			CredentialsDir: "n8n_credentials",
			IgnoreFields: []string{
				"createdAt", "updatedAt", "versionId", "activeVersionId",
				"versionCounter", "triggerCount", "tags", "shared", "description",
			},
		},
		Import: ImportConfig{
			MirrorDeletes: true,
			PublishActive: true,
		},
		Docker: DockerConfig{
			ComposeFile:       "docker-compose.yaml",
			AutoImportOnEmpty: true,
		},
		CredentialTypes: defaultCredentialTypes(),
	}
}

// ResolveDBConfig resolves env var references in a DBConfig, returning
// the final connection parameters. Env vars (the _env fields) take
// precedence over literal values.
func ResolveDBConfig(db DBConfig) (host string, port int, database, user, password string, err error) {
	host = db.Host
	if db.HostEnv != "" {
		host = os.Getenv(db.HostEnv)
		if host == "" {
			err = fmt.Errorf("env var %s is not set", db.HostEnv)
			return
		}
	}

	port = db.Port
	if db.PortEnv != "" {
		portStr := os.Getenv(db.PortEnv)
		if portStr == "" {
			err = fmt.Errorf("env var %s is not set", db.PortEnv)
			return
		}
		port, err = strconv.Atoi(portStr)
		if err != nil {
			err = fmt.Errorf("env var %s is not a valid port number: %s", db.PortEnv, portStr)
			return
		}
	}

	database = db.Database
	if db.DatabaseEnv != "" {
		database = os.Getenv(db.DatabaseEnv)
		if database == "" {
			err = fmt.Errorf("env var %s is not set", db.DatabaseEnv)
			return
		}
	}

	user = db.User
	if db.UserEnv != "" {
		user = os.Getenv(db.UserEnv)
		if user == "" {
			err = fmt.Errorf("env var %s is not set", db.UserEnv)
			return
		}
	}

	password = db.Password
	if db.PasswordEnv != "" {
		password = os.Getenv(db.PasswordEnv)
		if password == "" {
			err = fmt.Errorf("env var %s is not set", db.PasswordEnv)
			return
		}
	}

	return
}

// ResolveEncryptionKey resolves the encryption key from the environment.
func ResolveEncryptionKey(env Environment) (string, error) {
	if env.EncryptionKeyEnv == "" {
		return "", fmt.Errorf("encryption_key_env is not configured")
	}
	key := os.Getenv(env.EncryptionKeyEnv)
	if key == "" {
		return "", fmt.Errorf("env var %s is not set", env.EncryptionKeyEnv)
	}
	return key, nil
}

// ResolveEncryptionKeyList resolves the optional list of old encryption keys
// for key migration. Returns nil if not configured.
func ResolveEncryptionKeyList(env Environment) []string {
	if env.EncryptionKeyListEnv == "" {
		return nil
	}
	val := os.Getenv(env.EncryptionKeyListEnv)
	if val == "" {
		return nil
	}
	keys := strings.Split(val, ",")
	var result []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			result = append(result, k)
		}
	}
	return result
}

func defaultCredentialTypes() map[string]CredentialType {
	return map[string]CredentialType{
		"openAiApi": {
			Fields: []FieldDef{
				{Name: "apiKey", Secret: true, EnvSuffix: "_API_KEY"},
				{Name: "organizationId", Optional: true, EnvSuffix: "_ORGANIZATION_ID"},
				{Name: "url", Optional: true, EnvSuffix: "_URL"},
			},
		},
		"openRouterApi": {
			Fields: []FieldDef{
				{Name: "apiKey", Secret: true, EnvSuffix: "_API_KEY"},
			},
		},
		"httpHeaderAuth": {
			Fields: []FieldDef{
				{Name: "name"},
				{Name: "value", Secret: true},
			},
			Instances: map[string]InstanceOverride{
				"n8n_webhook_auth": {
					DisplayName: "N8N Webhook Auth",
					Overrides:   map[string]string{"name": "Authorization", "value_env": "N8N_WEBHOOK_AUTH"},
				},
				"supadata_account": {
					DisplayName: "Supadata Account",
					Overrides:   map[string]string{"name": "x-api-key", "value_env": "SUPADATA_API_KEY"},
				},
				"whapi_account": {
					DisplayName: "Whapi Account",
					Overrides:   map[string]string{"name": "Authorization", "value_env": "WHAPI_API_TOKEN", "value_transform": "bearer_prefix"},
				},
				"assembly_ai_account": {
					DisplayName: "Assembly AI Account",
					Overrides:   map[string]string{"name": "Authorization", "value_env": "ASSEMBLY_AI_API_KEY"},
				},
			},
		},
		"httpQueryAuth": {
			Fields: []FieldDef{
				{Name: "name"},
				{Name: "value", Secret: true},
			},
		},
		"supabaseApi": {
			Fields: []FieldDef{
				{Name: "host", Env: "SUPABASE_URL"},
				{Name: "serviceRole", Secret: true, Env: "SUPABASE_SERVICE_ROLE_KEY"},
			},
		},
		"youTubeOAuth2Api": {
			OAuth2: &OAuth2Config{
				TokenURL:     "https://oauth2.googleapis.com/token",
				AutoRefresh:  true,
				ScopeDefault: "https://www.googleapis.com/auth/youtube",
			},
			Fields: []FieldDef{
				{Name: "clientId", EnvSuffix: "_CLIENT_ID"},
				{Name: "clientSecret", Secret: true, EnvSuffix: "_CLIENT_SECRET"},
				{Name: "oauthTokenData.refresh_token", Secret: true, EnvSuffix: "_REFRESH_TOKEN"},
			},
		},
	}
}
