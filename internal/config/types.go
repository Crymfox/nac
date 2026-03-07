package config

// Config is the top-level nac.yaml configuration.
type Config struct {
	N8NVersion      string                    `yaml:"n8n_version" mapstructure:"n8n_version"`
	Environments    map[string]Environment    `yaml:"environments" mapstructure:"environments"`
	Export          ExportConfig              `yaml:"export" mapstructure:"export"`
	Import          ImportConfig              `yaml:"import" mapstructure:"import"`
	Docker          DockerConfig              `yaml:"docker" mapstructure:"docker"`
	CredentialTypes map[string]CredentialType `yaml:"credential_types" mapstructure:"credential_types"`
}

// Environment defines a target n8n database environment.
type Environment struct {
	DB                   DBConfig `yaml:"db" mapstructure:"db"`
	EncryptionKeyEnv     string   `yaml:"encryption_key_env" mapstructure:"encryption_key_env"`
	EncryptionKeyListEnv string   `yaml:"encryption_key_list_env,omitempty" mapstructure:"encryption_key_list_env"`
	APIKeyEnv            string   `yaml:"api_key_env,omitempty" mapstructure:"api_key_env"`
	APIUrlEnv            string   `yaml:"api_url_env,omitempty" mapstructure:"api_url_env"`
}

// DBConfig holds Postgres connection settings.
// Each field can be a literal value or an env var reference (suffix _env).
// When the _env variant is set, the value is resolved from the environment at runtime.
type DBConfig struct {
	// Literal values
	Host     string `yaml:"host,omitempty" mapstructure:"host"`
	Port     int    `yaml:"port,omitempty" mapstructure:"port"`
	Database string `yaml:"database,omitempty" mapstructure:"database"`
	User     string `yaml:"user,omitempty" mapstructure:"user"`
	Password string `yaml:"password,omitempty" mapstructure:"password"`

	// Env var references (take precedence over literals when set)
	HostEnv     string `yaml:"host_env,omitempty" mapstructure:"host_env"`
	PortEnv     string `yaml:"port_env,omitempty" mapstructure:"port_env"`
	DatabaseEnv string `yaml:"database_env,omitempty" mapstructure:"database_env"`
	UserEnv     string `yaml:"user_env,omitempty" mapstructure:"user_env"`
	PasswordEnv string `yaml:"password_env,omitempty" mapstructure:"password_env"`

	// SSL settings
	SSL                   bool `yaml:"ssl,omitempty" mapstructure:"ssl"`
	SSLRejectUnauthorized bool `yaml:"ssl_reject_unauthorized,omitempty" mapstructure:"ssl_reject_unauthorized"`
}

// ExportConfig controls how nac exports workflows and credentials.
type ExportConfig struct {
	WorkflowsDir   string   `yaml:"workflows_dir" mapstructure:"workflows_dir"`
	CredentialsDir string   `yaml:"credentials_dir" mapstructure:"credentials_dir"`
	IgnoreFields   []string `yaml:"ignore_fields" mapstructure:"ignore_fields"`
}

// ImportConfig controls how nac imports into a target environment.
type ImportConfig struct {
	MirrorDeletes bool `yaml:"mirror_deletes" mapstructure:"mirror_deletes"`
	PublishActive bool `yaml:"publish_active" mapstructure:"publish_active"`
}

// DockerConfig controls local Docker Compose behavior.
type DockerConfig struct {
	ComposeFile       string `yaml:"compose_file" mapstructure:"compose_file"`
	AutoImportOnEmpty bool   `yaml:"auto_import_on_empty" mapstructure:"auto_import_on_empty"`
}

// CredentialType defines how a credential type is built and exported.
type CredentialType struct {
	Fields    []FieldDef                  `yaml:"fields" mapstructure:"fields"`
	Instances map[string]InstanceOverride `yaml:"instances,omitempty" mapstructure:"instances"`
	OAuth2    *OAuth2Config               `yaml:"oauth2,omitempty" mapstructure:"oauth2"`
}

// FieldDef describes a single field in a credential's data JSON.
type FieldDef struct {
	Name      string `yaml:"name" mapstructure:"name"`
	Secret    bool   `yaml:"secret,omitempty" mapstructure:"secret"`
	Env       string `yaml:"env,omitempty" mapstructure:"env"`
	EnvSuffix string `yaml:"env_suffix,omitempty" mapstructure:"env_suffix"`
	Optional  bool   `yaml:"optional,omitempty" mapstructure:"optional"`
}

// InstanceOverride provides per-credential overrides for generic types like httpHeaderAuth.
type InstanceOverride struct {
	DisplayName string            `yaml:"display_name,omitempty" mapstructure:"display_name"`
	Overrides   map[string]string `yaml:"overrides,omitempty" mapstructure:"overrides"`
}

// OAuth2Config holds OAuth2-specific settings for credential types.
type OAuth2Config struct {
	TokenURL     string `yaml:"token_url" mapstructure:"token_url"`
	AutoRefresh  bool   `yaml:"auto_refresh" mapstructure:"auto_refresh"`
	ScopeDefault string `yaml:"scope_default,omitempty" mapstructure:"scope_default"`
}
