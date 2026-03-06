# nac - n8n As Code

> A Go CLI that turns n8n workflows and credentials into version-controlled, CI-deployable code.
> Zero runtime dependencies. Direct Postgres. No n8n CLI required.

---

## Table of Contents

- [Decision Log](#decision-log)
- [CLI Command Surface](#cli-command-surface)
- [Project Structure](#project-structure)
- [Config File Reference](#config-file-reference)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Project Foundation + Config](#phase-1-project-foundation--config)
  - [Phase 2: Database Layer + Crypto](#phase-2-database-layer--crypto)
  - [Phase 3: Workflow Export + Import](#phase-3-workflow-export--import)
  - [Phase 4: Credential Export + Import](#phase-4-credential-export--import)
  - [Phase 5: Local Dev Stack](#phase-5-local-dev-stack)
  - [Phase 6: CI Generation + API Tool](#phase-6-ci-generation--api-tool)
  - [Phase 7: Distribution + Polish](#phase-7-distribution--polish)
- [n8n Version Compatibility Strategy](#n8n-version-compatibility-strategy)
- [Schema Compatibility Agent](#schema-compatibility-agent)
- [Technical Risks + Mitigations](#technical-risks--mitigations)
- [Extraction Map](#extraction-map)
- [Timeline](#timeline)

---

## Decision Log

| Decision                 | Choice                             | Rationale                                                                                     |
| ------------------------ | ---------------------------------- | --------------------------------------------------------------------------------------------- |
| Language                 | **Go**                             | Single binary, zero runtime deps, cross-platform, fast startup. Perfect for CLI tooling.      |
| Credential type system   | **Config-driven (YAML)**           | Users define type mappings in `nac.yaml`. No code changes needed for new credential types.    |
| Distribution             | **CLI tool** (`brew`, `go install`) | Install globally, run `nac export`, `nac import`. Polished developer experience.              |
| Local dev stack          | **Included**                       | Provide a ready-to-use `docker-compose.yaml` as part of scaffolding. Complete local-to-prod.  |
| CI platform              | **GitHub Actions only**            | Most common. Provide a well-documented template workflow. Keep scope focused.                  |
| DB access                | **Direct Postgres via pgx**        | No Docker needed for export/import ops. Faster, works natively in CI. Docker only for local.  |
| n8n CLI dependency       | **Bypassed (pure SQL)**            | All import/export logic as direct SQL against `workflow_entity`/`credentials_entity`. Fully self-contained. |
| Binary name              | **`nac`**                          | Short, 3 chars to type. `nac init`, `nac export`, `nac import`.                               |
| n8n version              | **Pinned to 2.3.4** (current)     | Schema compatibility agent will handle future versions automatically (see below).             |

---

## CLI Command Surface

```
nac init                          # Scaffold project in current dir
nac export workflows              # Export workflows from DB to files
nac export credentials            # Export credentials from DB to files
nac import workflows              # Import workflows from files to DB
nac import credentials            # Import credentials from files to DB
nac up                            # Start local Docker Compose stack
nac down                          # Stop local Docker Compose stack
nac logs [service]                # Tail Docker Compose logs
nac api list-workflows            # n8n API: list all workflows
nac api list-executions <wf_id>   # n8n API: list executions
nac api get-execution <exec_id>   # n8n API: get execution details
nac api get-node <exec_id> <name> # n8n API: get node data
nac ci generate                   # Generate GitHub Actions workflow file
nac version                       # Print version + pinned n8n version
```

### Global Flags

| Flag              | Default     | Description                                   |
| ----------------- | ----------- | --------------------------------------------- |
| `--env <name>`    | `local`     | Target environment (local/dev/staging/prod)    |
| `--config <path>` | `nac.yaml`  | Path to config file                            |
| `--verbose`       | `false`     | Verbose logging output                         |
| `--dry-run`       | `false`     | Show what would change without modifying DB    |

---

## Project Structure

```
nac/
├── cmd/nac/
│   └── main.go                        # Entry point
├── internal/
│   ├── cmd/                            # Cobra commands
│   │   ├── root.go                     # Root command, global flags
│   │   ├── init.go                     # nac init
│   │   ├── export.go                   # nac export workflows|credentials
│   │   ├── import.go                   # nac import workflows|credentials
│   │   ├── up.go                       # nac up
│   │   ├── down.go                     # nac down
│   │   ├── logs.go                     # nac logs
│   │   ├── api.go                      # nac api subcommands
│   │   └── ci.go                       # nac ci generate
│   ├── config/                         # Config loading + validation
│   │   ├── config.go                   # Viper-based loader
│   │   ├── types.go                    # Struct definitions
│   │   └── validate.go                 # Schema validation
│   ├── db/                             # Postgres via pgx
│   │   ├── client.go                   # Connection pool, SSL/TLS, env resolution
│   │   ├── workflows.go               # workflow_entity queries
│   │   └── credentials.go             # credentials_entity queries
│   ├── crypto/                         # AES-256-CBC (OpenSSL compat)
│   │   ├── encrypt.go                  # Encrypt with Salted__ prefix
│   │   ├── decrypt.go                  # Decrypt with MD5 key derivation
│   │   └── crypto_test.go             # Round-trip + known-value tests
│   ├── workflow/                       # Workflow business logic
│   │   ├── export.go                   # Split, diff, normalize
│   │   ├── import.go                   # Aggregate, remap, upsert, mirror
│   │   ├── remap.go                    # executeWorkflow reference remapping
│   │   ├── publish.go                  # Publish active workflows (n8n 2.x)
│   │   └── sanitize.go                # Folder name sanitization
│   ├── credential/                     # Credential business logic
│   │   ├── export.go                   # Decrypt, structural extract, placeholders
│   │   ├── import.go                   # Build from env, encrypt, upsert, mirror
│   │   ├── builder.go                  # Config-driven credential data builder
│   │   ├── registry.go                 # Type registry from nac.yaml
│   │   ├── oauth2.go                   # OAuth2 token refresh
│   │   └── transforms.go              # Value transforms (bearer_prefix, etc.)
│   ├── docker/                         # Docker Compose management
│   │   └── compose.go                  # up, down, logs, network detection
│   ├── n8napi/                         # n8n REST API client
│   │   ├── client.go                   # HTTP client with API key auth
│   │   ├── workflows.go               # Workflow endpoints
│   │   ├── executions.go              # Execution endpoints
│   │   └── types.go                   # API response types
│   └── output/                         # Output formatting
│       ├── table.go                    # Table format
│       ├── json.go                     # JSON format
│       └── summary.go                  # Summary format
├── templates/                          # go:embed scaffolding templates
│   ├── nac.yaml.tmpl                   # Default config
│   ├── docker-compose.yaml.tmpl        # Local dev stack
│   ├── env.local.example.tmpl          # Local env template
│   ├── env.remote.example.tmpl         # Remote env template
│   ├── github-actions.yaml.tmpl        # CI pipeline
│   └── gitignore.tmpl                  # .gitignore additions
├── schema/                             # n8n DB schema tracking
│   └── 2.3.4.sql                       # Baseline schema snapshot
├── go.mod
├── go.sum
├── .goreleaser.yml                     # Cross-platform builds
├── Makefile                            # Dev tasks
├── LICENSE                             # MIT
└── README.md
```

---

## Config File Reference

The `nac.yaml` file is the single source of truth for a project. Generated by `nac init`.

```yaml
# =============================================================================
# nac.yaml - n8n As Code configuration
# =============================================================================

# Pinned n8n version. nac's SQL queries and crypto are validated against
# this version. The schema compatibility agent will notify when this
# needs updating.
n8n_version: "2.3.4"

# =============================================================================
# Environments
# =============================================================================
# Each environment maps to a Postgres database.
# Values can be literals or env var references (suffix _env).
# The active environment is selected via --env flag (default: local).

environments:
  local:
    db:
      host: localhost
      port: 5432
      database: n8n
      user: n8n
      password: n8n
      ssl: false
    encryption_key_env: N8N_ENCRYPTION_KEY
    # Optional: list of old keys for re-encryption migration
    encryption_key_list_env: N8N_ENCRYPTION_KEY_LIST

  dev:
    db:
      host_env: DB_POSTGRESDB_HOST
      port_env: DB_POSTGRESDB_PORT
      database_env: DB_POSTGRESDB_DATABASE
      user_env: DB_POSTGRESDB_USER
      password_env: DB_POSTGRESDB_PASSWORD
      ssl: true
      ssl_reject_unauthorized: false
    encryption_key_env: N8N_ENCRYPTION_KEY

  staging:
    db:
      host_env: DB_POSTGRESDB_HOST
      port_env: DB_POSTGRESDB_PORT
      database_env: DB_POSTGRESDB_DATABASE
      user_env: DB_POSTGRESDB_USER
      password_env: DB_POSTGRESDB_PASSWORD
      ssl: true
    encryption_key_env: N8N_ENCRYPTION_KEY

  production:
    db:
      host_env: DB_POSTGRESDB_HOST
      port_env: DB_POSTGRESDB_PORT
      database_env: DB_POSTGRESDB_DATABASE
      user_env: DB_POSTGRESDB_USER
      password_env: DB_POSTGRESDB_PASSWORD
      ssl: true
    encryption_key_env: N8N_ENCRYPTION_KEY

# =============================================================================
# Export settings
# =============================================================================

export:
  workflows_dir: n8n_workflows
  credentials_dir: n8n_credentials

  # Fields stripped before diffing to avoid noisy Git changes.
  # These are instance-specific metadata that differ across environments.
  ignore_fields:
    - createdAt
    - updatedAt
    - versionId
    - activeVersionId
    - versionCounter
    - triggerCount
    - tags
    - shared
    - description

# =============================================================================
# Import settings
# =============================================================================

import:
  # Delete resources in remote DB that don't exist in the repo.
  # When true, the repo becomes the single source of truth.
  mirror_deletes: true

  # For n8n 2.x: explicitly publish workflows marked as active.
  # Required because n8n 2.x has a separate "published" state.
  publish_active: true

# =============================================================================
# Docker Compose (local dev stack)
# =============================================================================

docker:
  compose_file: docker-compose.yaml
  # Auto-import workflows + credentials when local DB is empty on `nac up`
  auto_import_on_empty: true

# =============================================================================
# Credential Type Definitions
# =============================================================================
# Config-driven system for building credential data at import time.
# Each type defines:
#   - display_name: How the credential appears in n8n UI
#   - fields: List of data fields with env var mapping
#   - instances: Per-folder overrides (for types like httpHeaderAuth
#     where different credentials of the same type need different fixed values)
#   - oauth2: OAuth2-specific settings (token refresh)
#
# Field properties:
#   - name: JSON field name in the credential data
#   - secret: true if this field contains sensitive data (stripped on export)
#   - env: Exact env var name (takes precedence over env_suffix)
#   - env_suffix: Appended to FOLDER_NAME_UPPERCASE (e.g., _API_KEY)
#   - optional: true if the field may be absent
#
# Value transforms (applied to resolved env var value):
#   - bearer_prefix: Prepend "Bearer " if not already present

credential_types:
  # --- API Key credentials ---

  openAiApi:
    fields:
      - name: apiKey
        secret: true
        env_suffix: _API_KEY
      - name: organizationId
        optional: true
        env_suffix: _ORGANIZATION_ID
      - name: url
        optional: true
        env_suffix: _URL

  openRouterApi:
    fields:
      - name: apiKey
        secret: true
        env_suffix: _API_KEY

  # --- Header/Query auth credentials ---
  # These need per-instance overrides because the header name is fixed
  # per credential, but the type itself is generic.

  httpHeaderAuth:
    fields:
      - name: name
      - name: value
        secret: true
    instances:
      n8n_webhook_auth:
        display_name: "N8N Webhook Auth"
        overrides:
          name: "Authorization"
          value_env: N8N_WEBHOOK_AUTH
      supadata_account:
        display_name: "Supadata Account"
        overrides:
          name: "x-api-key"
          value_env: SUPADATA_API_KEY
      whapi_account:
        display_name: "Whapi Account"
        overrides:
          name: "Authorization"
          value_env: WHAPI_API_TOKEN
          value_transform: bearer_prefix
      assembly_ai_account:
        display_name: "Assembly AI Account"
        overrides:
          name: "Authorization"
          value_env: ASSEMBLY_AI_API_KEY

  httpQueryAuth:
    fields:
      - name: name
      - name: value
        secret: true

  # --- Service-specific credentials ---

  supabaseApi:
    fields:
      - name: host
        env: SUPABASE_URL
      - name: serviceRole
        secret: true
        env: SUPABASE_SERVICE_ROLE_KEY

  # --- OAuth2 credentials ---

  youTubeOAuth2Api:
    oauth2:
      token_url: https://oauth2.googleapis.com/token
      auto_refresh: true
      scope_default: "https://www.googleapis.com/auth/youtube"
    fields:
      - name: clientId
        env_suffix: _CLIENT_ID
      - name: clientSecret
        secret: true
        env_suffix: _CLIENT_SECRET
      - name: oauthTokenData.refresh_token
        secret: true
        env_suffix: _REFRESH_TOKEN
```

---

## Implementation Phases

### Phase 1: Project Foundation + Config

**Estimated effort: 2-3 days**

| Task                    | Details                                                                                            |
| ----------------------- | -------------------------------------------------------------------------------------------------- |
| Go module init          | `github.com/crymfox/nac`, Go 1.22+                                                              |
| Cobra CLI skeleton      | Root command with global flags (`--env`, `--config`, `--verbose`, `--dry-run`), version subcommand |
| Config system           | Viper-based loader for `nac.yaml`. Struct definitions with validation tags. Env var resolution.    |
| `nac init` command      | Interactive scaffolding using `go:embed` templates. Creates `nac.yaml`, `docker-compose.yaml`, `.env.local.example`, `.env.remote.example`, `.gitignore`, workflow/credential directories |
| Unit tests              | Config loading, validation, template rendering, env var resolution                                 |

**Go dependencies added:**

| Package                    | Purpose                |
| -------------------------- | ---------------------- |
| `github.com/spf13/cobra`  | CLI framework          |
| `github.com/spf13/viper`  | Config loading         |

**Acceptance criteria:**
- `nac init` creates a working project skeleton
- `nac version` prints version + pinned n8n version
- Config loading handles all env var resolution patterns (`_env` suffix)
- Validation catches missing required fields

---

### Phase 2: Database Layer + Crypto

**Estimated effort: 2-3 days**

| Task                       | Details                                                                                                                   |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| Postgres client            | pgx-based client with connection pooling, SSL/TLS support (with `ssl_reject_unauthorized` option), env-based config resolution |
| AES-256-CBC crypto         | Encrypt/decrypt matching n8n's OpenSSL format: `Salted__` 8-byte prefix, MD5 key derivation (`EVP_BytesToKey`), base64 encoding. **This is the most critical component.** |
| DB schema queries          | `workflow_entity`: SELECT all, SELECT name-to-id map, INSERT/UPDATE (upsert), DELETE by name. `credentials_entity`: identical query set |
| Schema snapshot            | Dump and store the n8n 2.3.4 schema in `schema/2.3.4.sql` as baseline for the compatibility agent                        |
| Tests                      | Crypto round-trip tests against known n8n-encrypted values from the current project. DB integration tests using testcontainers-go |

**Go dependencies added:**

| Package                       | Purpose                       |
| ----------------------------- | ----------------------------- |
| `github.com/jackc/pgx/v5`    | Postgres driver (direct, no Docker) |
| `crypto/aes`, `crypto/cipher` | AES-256-CBC (stdlib)          |
| `crypto/md5`                  | Key derivation (stdlib)       |

**Critical: AES-256-CBC implementation**

n8n encrypts credential data using Node.js `crypto.createCipheriv` which produces OpenSSL-compatible output:

```
Salted__ (8 bytes) + salt (8 bytes) + ciphertext
```

Key derivation uses `EVP_BytesToKey` with MD5. The Go implementation must produce byte-identical output to n8n's Node.js implementation. Test against actual encrypted credential data from the existing project.

```go
// Pseudocode for the encryption path:
func Encrypt(plaintext []byte, passphrase string) (string, error) {
    salt := randomBytes(8)
    key, iv := evpBytesToKey(passphrase, salt, 32, 16) // AES-256 = 32-byte key, 16-byte IV
    ciphertext := aesCBCEncrypt(key, iv, pkcs7Pad(plaintext))
    result := append([]byte("Salted__"), salt...)
    result = append(result, ciphertext...)
    return base64.StdEncoding.EncodeToString(result), nil
}
```

**Acceptance criteria:**
- Can decrypt actual n8n credentials exported from the current project
- Can encrypt data and have n8n successfully decrypt it
- Connection pooling works with both local (no SSL) and remote (SSL) Postgres
- All CRUD queries work against a real n8n 2.3.4 database

---

### Phase 3: Workflow Export + Import

**Estimated effort: 3-4 days**

#### Export (`nac export workflows`)

| Step | Logic                                                                                              |
| ---- | -------------------------------------------------------------------------------------------------- |
| 1    | Connect to DB via pgx, `SELECT * FROM workflow_entity`                                             |
| 2    | For each workflow, sanitize name to folder: lowercase, preserve parentheses, non-alnum to `_`      |
| 3    | Normalize: set `active = false` when missing                                                       |
| 4    | Smart diff: strip `ignore_fields` from both new and existing file, compare structurally            |
| 5    | If changed, write full workflow JSON to `<workflows_dir>/<folder>/workflow.json`                   |
| 6    | Track exported folders. Remove stale folders not present in DB.                                    |
| 7    | Print summary: `Updated: N`, `Unchanged: N`, `Removed: N`                                         |

#### Import (`nac import workflows`)

| Step | Logic                                                                                              |
| ---- | -------------------------------------------------------------------------------------------------- |
| 1    | Find all `workflow.json` files under `<workflows_dir>/`                                            |
| 2    | Aggregate into a list, normalize fields (`active`, `isArchived` defaults)                          |
| 3    | Fetch remote name-to-id map: `SELECT name, id FROM workflow_entity`                                |
| 4    | Remap incoming `.id` to match existing remote IDs by name (upsert-by-name)                         |
| 5    | Build local id-to-name map for reverse lookup                                                      |
| 6    | Remap `executeWorkflow` node references: resolve via `cachedResultName` or local id-to-name, then map to remote IDs |
| 7    | Mirror deletes: `DELETE FROM workflow_entity WHERE name = $1` for names in DB but not in repo      |
| 8    | UPSERT each workflow into `workflow_entity`                                                        |
| 9    | Enforce `active`/`isArchived` flags: `UPDATE workflow_entity SET active = $1, "isArchived" = $2 WHERE name = $3` |
| 10   | Publish active workflows (see below)                                                               |
| 11   | Print summary: `Imported: N`, `Deleted: N`, `Published: N`                                        |

#### Publish Logic (n8n 2.x)

n8n 2.x requires workflows to be explicitly "published" to be live. Options:

1. **Primary: Direct SQL** - Investigate what `n8n publish:workflow --id=X` does to the DB. If it's a version table insert or a flag update, replicate it in SQL.
2. **Fallback: n8n REST API** - If the instance is reachable, `PATCH /api/v1/workflows/{id}` with `{ "active": true }`.
3. **Last resort: Docker** - Run `n8n publish:workflow` via Docker container (only for edge cases).

**Acceptance criteria:**
- Idempotent: running export twice produces no Git diff
- ID remapping is correct across environments
- `executeWorkflow` references resolve correctly by name
- Mirror deletes only remove workflows not in the repo
- Active workflows are live after import

---

### Phase 4: Credential Export + Import

**Estimated effort: 3-4 days**

#### Config-Driven Builder

The builder reads `credential_types` from `nac.yaml` and creates a registry:

```go
type CredentialBuilder struct {
    Type       string
    Fields     []FieldDef
    Instances  map[string]InstanceOverride
    OAuth2     *OAuth2Config
}

// BuildData resolves env vars and produces the JSON credential data
func (b *CredentialBuilder) BuildData(folderName string) ([]byte, error)

// ExtractStructural returns only non-secret fields (for diff comparison)
func (b *CredentialBuilder) ExtractStructural(data []byte) ([]byte, error)

// ReplaceSecrets returns data with secrets replaced by ENV: placeholders
func (b *CredentialBuilder) ReplaceSecrets(data []byte, folderName string) ([]byte, error)
```

#### Export (`nac export credentials`)

| Step | Logic                                                                                              |
| ---- | -------------------------------------------------------------------------------------------------- |
| 1    | Connect to DB, `SELECT * FROM credentials_entity`                                                  |
| 2    | For each credential, decrypt `data` field using AES-256-CBC + `N8N_ENCRYPTION_KEY`                |
| 3    | Look up type in builder registry                                                                   |
| 4    | Extract structural (non-secret) fields for comparison                                              |
| 5    | Smart diff against existing file (compare id, name, type, structural fields)                       |
| 6    | If changed, write minimal JSON with `ENV:FOLDER_NAME` placeholder in data field                    |
| 7    | Remove stale folders. Print summary.                                                               |

#### Import (`nac import credentials`)

| Step | Logic                                                                                              |
| ---- | -------------------------------------------------------------------------------------------------- |
| 1    | Find all `credential.json` files under `<credentials_dir>/`                                        |
| 2    | For each file, read type and `ENV:` placeholder from data field                                    |
| 3    | Look up type in builder registry. Resolve instance overrides if applicable.                        |
| 4    | Build credential data JSON from environment variables                                              |
| 5    | For OAuth2 types with `auto_refresh: true`: call token endpoint to get fresh access_token          |
| 6    | Apply value transforms (`bearer_prefix`, etc.)                                                     |
| 7    | Encrypt built JSON with target environment's `N8N_ENCRYPTION_KEY`                                  |
| 8    | Derive display name from folder name (with special-case mappings)                                  |
| 9    | Fetch remote name-to-id map, remap IDs (upsert-by-name)                                           |
| 10   | Mirror deletes for credentials in DB but not in repo                                               |
| 11   | UPSERT each credential into `credentials_entity`                                                   |
| 12   | Print summary.                                                                                      |

#### Encryption Key Migration

If `N8N_ENCRYPTION_KEY_LIST` is set (comma-separated list of old keys):

1. After import, iterate each credential in the DB
2. Try to decrypt `data` with each old key from the list
3. On successful decrypt, re-encrypt with the current `N8N_ENCRYPTION_KEY`
4. Update the row in `credentials_entity`
5. Log which credentials were migrated, which failed

**Acceptance criteria:**
- Config-driven builder handles all 6 credential types from the current project
- Adding a new credential type requires only `nac.yaml` changes (no Go code)
- `httpHeaderAuth` per-instance overrides work correctly
- OAuth2 token refresh works for YouTube credentials
- Encryption key migration works end-to-end
- Unknown credential types produce clear error messages

---

### Phase 5: Local Dev Stack

**Estimated effort: 1-2 days**

| Task                        | Details                                                                                          |
| --------------------------- | ------------------------------------------------------------------------------------------------ |
| `nac up`                    | Run `docker compose -f <compose_file> up -d`. Wait for Postgres readiness (TCP probe on port). If `auto_import_on_empty` is true and DB has no workflows, auto-run `nac import workflows` + `nac import credentials`. |
| `nac down`                  | Run `docker compose -f <compose_file> down`                                                      |
| `nac logs [service]`        | Run `docker compose -f <compose_file> logs -f [service]`                                         |
| Docker Compose template     | Generated by `nac init`. Configurable: n8n image tag (from `n8n_version`), Postgres version, Redis version, worker on/off, memory limits. New Relic integration optional. |
| Network detection           | Auto-detect Docker network name from running containers (same logic as current bash scripts)     |

**Docker Compose template features:**
- 4 services: `postgres:15`, `redis:7-alpine`, `n8n-primary`, `n8n-worker`
- Queue-based execution mode with Redis/Bull
- Task runners enabled
- Env var passthrough for all n8n config
- Volume mounts for data persistence
- `extra_hosts` for `host.docker.internal` access

**Acceptance criteria:**
- `nac up` starts a fully working local n8n instance
- Auto-import populates an empty DB with workflows + credentials from the repo
- `nac down` cleanly stops everything
- `nac logs n8n-primary` streams logs

---

### Phase 6: CI Generation + API Tool

**Estimated effort: 2-3 days**

#### `nac ci generate`

Generates `.github/workflows/deploy-n8n.yml` with:

```yaml
name: Deploy n8n

on:
  push:
    branches: [develop, main]
    paths:
      - "n8n_workflows/**/workflow.json"
      - "n8n_credentials/**/credential.json"

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: ${{ github.ref_name == 'main' && 'production' || 'develop' }}
    steps:
      - uses: actions/checkout@v4

      - name: Install nac
        run: go install github.com/crymfox/nac/cmd/nac@latest

      - name: Backup remote DB
        run: nac backup --env ${{ github.ref_name == 'main' && 'production' || 'dev' }}

      - name: Upload backup artifact
        uses: actions/upload-artifact@v4
        with:
          name: n8n-backup-${{ github.run_id }}
          path: backups/*.sql
          retention-days: 14

      - name: Import workflows
        run: nac import workflows --env ${{ ... }}

      - name: Import credentials
        run: nac import credentials --env ${{ ... }}
```

The template is parameterized based on `nac.yaml` environments.

#### `nac api` Subcommands

Thin HTTP client wrapping the n8n REST API v1:

| Command                                          | Description                            |
| ------------------------------------------------ | -------------------------------------- |
| `nac api list-workflows`                         | List all workflows (id, name, active)  |
| `nac api list-executions <workflow_id>`           | List executions for a workflow         |
| `nac api list-all-executions`                     | List executions across all workflows   |
| `nac api list-execution-nodes <execution_id>`     | List nodes with status in an execution |
| `nac api get-execution <id> [--format=json]`      | Full execution details                 |
| `nac api get-node <execution_id> <node_name>`     | Specific node input/output/error       |
| `nac api export-node-inputs <exec_id> <node>`     | Export node inputs to JSON file        |
| `nac api export-source-output <exec_id> <node>`   | Export upstream node's output          |
| `nac api extract-node-code <wf_id> <node>`        | Extract TypeScript code from Code node |

Configuration: `N8N_API_KEY` env var + `N8N_API_URL` (defaults to `http://localhost:5678/api/v1`).

Output formats: `--format=table` (default), `--format=json`, `--format=summary`.

**Acceptance criteria:**
- Generated CI workflow works on a real GitHub Actions run
- API commands produce clean, readable output
- API commands work both locally (Docker network) and against remote instances

---

### Phase 7: Distribution + Polish

**Estimated effort: 2-3 days**

| Task                | Details                                                                                            |
| ------------------- | -------------------------------------------------------------------------------------------------- |
| GoReleaser          | `.goreleaser.yml` for cross-platform builds: `linux/darwin/windows` x `amd64/arm64`                |
| Homebrew            | Tap formula at `crymfox/homebrew-tap`. `brew install crymfox/tap/nac`                          |
| `go install`        | Ensure `go install github.com/crymfox/nac/cmd/nac@latest` works                                 |
| GitHub Actions CI   | Test + lint + build pipeline for the nac repo itself. Matrix: Go versions x OS.                    |
| Release automation  | Tag-triggered releases via GoReleaser. Auto-generate changelog from conventional commits.          |
| README              | Features, quickstart guide, config reference, architecture diagram, comparison with alternatives   |
| LICENSE             | MIT                                                                                                |
| Contributing guide  | How to add new credential types (config-only), how to contribute code                              |

---

## n8n Version Compatibility Strategy

### Current Approach (v1)

Pin to n8n **2.3.4**. All SQL queries, the crypto implementation, and the publish logic are validated against this version. The `n8n_version` field in `nac.yaml` serves as documentation and is checked at runtime to warn users if they're running a different n8n version.

### Tables We Touch

`nac` only interacts with two core tables:

| Table                  | Operations                                    |
| ---------------------- | --------------------------------------------- |
| `workflow_entity`      | SELECT, INSERT, UPDATE, DELETE                |
| `credentials_entity`   | SELECT, INSERT, UPDATE, DELETE                |

Plus potentially version/publish-related tables for the publish workflow logic (to be determined during Phase 3).

### What Could Break Across n8n Versions

| Surface              | Risk Level | Example                                              |
| -------------------- | ---------- | ---------------------------------------------------- |
| Table names          | Low        | `workflow_entity` renamed to `workflows`             |
| Column names         | Medium     | `isArchived` renamed or removed                      |
| Column types         | Medium     | `id` changes from string to UUID                     |
| New required columns | Medium     | New non-nullable column with no default              |
| Encryption format    | High       | n8n switches from AES-256-CBC to a different cipher  |
| Publish mechanism    | Medium     | Version table schema changes in a major release      |

---

## Schema Compatibility Agent

### Overview

A GitHub Actions workflow that runs on a schedule (weekly) or on-demand. It spins up the **latest n8n version**, lets it initialize its DB schema, and diffs it against the pinned baseline. If breaking changes are detected, it auto-creates a GitHub issue (or PR) with details.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  GitHub Actions Runner                   │
│                                                         │
│  1. Pull latest n8n Docker image                        │
│  2. Start Postgres + n8n (init-only, then stop)         │
│  3. pg_dump the initialized schema                      │
│  4. Diff against schema/<pinned_version>.sql            │
│  5. Analyze diff for breaking changes                   │
│  6. If breaking: create GitHub Issue with details        │
│  7. If compatible: update schema/<new_version>.sql       │
│     and open PR to bump n8n_version                     │
└─────────────────────────────────────────────────────────┘
```

### Implementation

```yaml
# .github/workflows/schema-compat-check.yml
name: n8n Schema Compatibility Check

on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM UTC
  workflow_dispatch:       # Manual trigger

jobs:
  check-schema:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Get latest n8n version
        id: latest
        run: |
          LATEST=$(curl -s https://registry.hub.docker.com/v2/repositories/n8nio/n8n/tags/?page_size=10 \
            | jq -r '.results[].name' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1)
          PINNED=$(grep 'n8n_version' schema/PINNED_VERSION || echo "2.3.4")
          echo "latest=$LATEST" >> $GITHUB_OUTPUT
          echo "pinned=$PINNED" >> $GITHUB_OUTPUT

      - name: Skip if already pinned to latest
        if: steps.latest.outputs.latest == steps.latest.outputs.pinned
        run: echo "Already on latest version" && exit 0

      - name: Start Postgres
        run: |
          docker run -d --name pg \
            -e POSTGRES_USER=n8n -e POSTGRES_PASSWORD=n8n -e POSTGRES_DB=n8n \
            -p 5432:5432 postgres:15
          sleep 5

      - name: Initialize n8n schema (latest version)
        run: |
          docker run --rm --network host \
            -e DB_TYPE=postgresdb \
            -e DB_POSTGRESDB_HOST=localhost \
            -e DB_POSTGRESDB_PORT=5432 \
            -e DB_POSTGRESDB_DATABASE=n8n \
            -e DB_POSTGRESDB_USER=n8n \
            -e DB_POSTGRESDB_PASSWORD=n8n \
            -e N8N_ENCRYPTION_KEY=test_key_for_schema_check \
            n8nio/n8n:${{ steps.latest.outputs.latest }} \
            n8n start --tunnel &
          sleep 30 && docker stop $(docker ps -q --filter ancestor=n8nio/n8n:${{ steps.latest.outputs.latest }})

      - name: Dump new schema
        run: |
          docker exec pg pg_dump -U n8n -d n8n --schema-only \
            --table=workflow_entity --table=credentials_entity \
            > schema/new.sql

      - name: Diff schemas
        id: diff
        run: |
          DIFF=$(diff schema/${{ steps.latest.outputs.pinned }}.sql schema/new.sql || true)
          if [ -z "$DIFF" ]; then
            echo "status=compatible" >> $GITHUB_OUTPUT
          else
            echo "status=changed" >> $GITHUB_OUTPUT
            echo "$DIFF" > schema/diff.txt
          fi

      - name: Analyze breaking changes
        if: steps.diff.outputs.status == 'changed'
        id: analyze
        run: |
          # Check for breaking patterns in the diff
          BREAKING=false

          # Table renames or drops
          if grep -q 'DROP TABLE\|ALTER TABLE.*RENAME' schema/diff.txt; then
            BREAKING=true
          fi

          # Column renames, type changes, or drops on our tables
          if grep -q 'DROP COLUMN\|ALTER COLUMN.*TYPE\|RENAME COLUMN' schema/diff.txt; then
            BREAKING=true
          fi

          # Changes to columns we use
          for col in id name nodes connections settings active '"isArchived"' data type; do
            if grep -q "$col" schema/diff.txt; then
              BREAKING=true
              break
            fi
          done

          echo "breaking=$BREAKING" >> $GITHUB_OUTPUT

      - name: Create issue for breaking changes
        if: steps.analyze.outputs.breaking == 'true'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const diff = fs.readFileSync('schema/diff.txt', 'utf8');
            const latest = '${{ steps.latest.outputs.latest }}';
            const pinned = '${{ steps.latest.outputs.pinned }}';

            await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: `[Schema Agent] Breaking n8n schema changes detected: ${pinned} -> ${latest}`,
              labels: ['schema-compat', 'breaking-change'],
              body: `## n8n Schema Change Detected\n\n` +
                    `| | Version |\n|---|---|\n` +
                    `| Pinned | ${pinned} |\n` +
                    `| Latest | ${latest} |\n\n` +
                    `### Schema Diff\n\n\`\`\`diff\n${diff}\n\`\`\`\n\n` +
                    `### Action Required\n\n` +
                    `1. Review the schema changes above\n` +
                    `2. Update SQL queries in \`internal/db/\` if needed\n` +
                    `3. Update \`schema/${latest}.sql\` with the new baseline\n` +
                    `4. Bump \`n8n_version\` in templates and tests\n` +
                    `5. Run integration tests against n8n ${latest}\n`
            });

      - name: Create PR for compatible changes
        if: steps.diff.outputs.status == 'changed' && steps.analyze.outputs.breaking == 'false'
        run: |
          LATEST="${{ steps.latest.outputs.latest }}"
          cp schema/new.sql "schema/${LATEST}.sql"
          git checkout -b "schema/bump-n8n-${LATEST}"
          echo "$LATEST" > schema/PINNED_VERSION
          git add schema/
          git commit -m "chore: bump n8n schema baseline to ${LATEST}"
          git push -u origin "schema/bump-n8n-${LATEST}"
          gh pr create \
            --title "chore: bump n8n version to ${LATEST}" \
            --body "Automated schema compatibility check found no breaking changes between the pinned version and n8n ${LATEST}. This PR updates the schema baseline." \
            --label "schema-compat"

      - name: No changes
        if: steps.diff.outputs.status == 'compatible'
        run: echo "Schema unchanged between versions. Nothing to do."
```

### What the Agent Checks

| Check                            | Action on Failure                                    |
| -------------------------------- | ---------------------------------------------------- |
| `workflow_entity` table exists   | Issue: "table renamed or dropped"                    |
| `credentials_entity` table exists| Issue: "table renamed or dropped"                    |
| Column names we use              | Issue: "column renamed/dropped"                      |
| Column types we use              | Issue: "column type changed"                         |
| New NOT NULL columns (no default)| Issue: "new required column, INSERT will fail"       |
| Schema additions (new columns)   | Auto-PR: "compatible addition, bump version"         |
| No changes at all                | No action                                            |

### Future Enhancements for the Agent

1. **Auto-fix simple changes** - If a column is renamed but the pattern is obvious (e.g., `isArchived` -> `is_archived`), auto-generate a migration PR with updated SQL queries.
2. **Encryption format detection** - Start up n8n, create a test credential, dump it, verify our crypto implementation can decrypt it.
3. **Multi-version matrix** - Test against last 3 minor versions to establish a support window.
4. **Slack/Discord notification** - Notify the team when breaking changes are detected.

---

## Technical Risks + Mitigations

| Risk | Impact | Likelihood | Mitigation |
| ---- | ------ | ---------- | ---------- |
| AES-256-CBC Go implementation doesn't match n8n's Node.js crypto byte-for-byte | Credentials will be unreadable after import. Data loss potential. | Medium | Test against real encrypted values from the current project. n8n uses OpenSSL's `Salted__` format with `EVP_BytesToKey` (MD5). This is well-documented. Go's stdlib has the primitives; just need the right glue. |
| `publish:workflow` touches unknown tables in n8n 2.x | Active workflows won't be "live" after import | Medium | Phase 3: investigate by diffing DB state before/after a publish. Fallback 1: n8n REST API `PATCH /workflows/{id}`. Fallback 2: Docker-based `n8n publish:workflow`. |
| n8n DB schema changes in future versions | SQL queries break silently or noisily | Medium | Schema compatibility agent (above). Pin version. Runtime version check with warning. |
| `httpHeaderAuth` per-instance overrides are brittle | Adding a new httpHeaderAuth credential requires config changes | Low | Document clearly. The instances pattern handles real-world cases. Unknown instances fall through to generic env var resolution (`FOLDER_NAME_VALUE`). |
| OAuth2 token refresh fails (expired refresh token, revoked access) | YouTube credential import fails | Medium | Clear error messages. Retry logic. Link to Google OAuth consent flow in error output. |
| Docker Compose version differences across platforms | `nac up` fails on older Docker versions | Low | Require Docker Compose V2 (the `docker compose` plugin). Document minimum version. |

---

## Extraction Map

What gets extracted from the current project and where it maps to in `nac`:

| Current File                                         | Maps To                                           |
| ---------------------------------------------------- | ------------------------------------------------- |
| `n8n/hack/export_local_workflows.sh`                | `internal/workflow/export.go`                     |
| `n8n/hack/export_local_credentials.sh`               | `internal/credential/export.go`                   |
| `n8n/hack/import_workflows_remote.sh`                | `internal/workflow/import.go` + `remap.go` + `publish.go` |
| `n8n/hack/import_creds_remote.sh`                    | `internal/credential/import.go` + `builder.go` + `oauth2.go` |
| `n8n/hack/n8n_api.sh`                                | `internal/n8napi/client.go` + `internal/cmd/api.go` |
| `n8n/docker-compose.yaml`                            | `templates/docker-compose.yaml.tmpl`              |
| `n8n/env.local.example`                              | `templates/env.local.example.tmpl`                |
| `n8n/.env.remote.*`                                  | `templates/env.remote.example.tmpl`               |
| `.github/workflows/push-n8n-wfs.yaml`                | `templates/github-actions.yaml.tmpl`              |
| `n8n/README.md`                                      | New standalone `README.md`                        |
| `n8n/N8N_API_TROUBLESHOOTING.md`                     | Part of new `README.md` or separate doc           |
| `n8n/docs/guide.md`                                  | Part of new `README.md` or separate doc           |
| Credential type mappings (hardcoded in bash scripts) | `credential_types` section in `nac.yaml`          |
| Folder name sanitization logic                       | `internal/workflow/sanitize.go`                   |
| Docker network detection logic                       | `internal/docker/compose.go`                      |

---

## Timeline

| Phase | Description                    | Effort     | Cumulative  |
| ----- | ------------------------------ | ---------- | ----------- |
| 1     | Foundation + Config            | 2-3 days   | 2-3 days    |
| 2     | Database + Crypto              | 2-3 days   | 4-6 days    |
| 3     | Workflow Export + Import       | 3-4 days   | 7-10 days   |
| 4     | Credential Export + Import     | 3-4 days   | 10-14 days  |
| 5     | Local Dev Stack                | 1-2 days   | 11-16 days  |
| 6     | CI Generation + API Tool       | 2-3 days   | 13-19 days  |
| 7     | Distribution + Polish          | 2-3 days   | 15-22 days  |
| -     | Schema Compatibility Agent     | 1-2 days   | 16-24 days  |
| **Total** |                            |            | **~3-4 weeks** |

### Suggested Phase Order

Phases 1-4 are sequential (each depends on the previous). Phases 5, 6, and 7 can be parallelized or reordered.

Recommended order for fastest time-to-value:

1. **Phase 1** (Foundation) - must be first
2. **Phase 2** (DB + Crypto) - must be second (everything depends on it)
3. **Phase 3** (Workflows) - core value, prove the architecture works
4. **Phase 4** (Credentials) - core value, completes the main functionality
5. **Phase 5** (Local Dev) - quick win, makes dogfooding easy
6. **Phase 7** (Distribution) - get it installable before adding more features
7. **Phase 6** (CI + API) - polish, can ship without this initially
8. **Schema Agent** - post-v1, run alongside ongoing development
