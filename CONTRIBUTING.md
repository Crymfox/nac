# Contributing to nac

First off, thank you for considering contributing to `nac`! It's people like you that make `nac` such a great tool for the n8n community.

## 🏗 Development Setup

### Prerequisites
- [Go 1.24+](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/) (required for integration tests)

### Getting Started
1. Fork the repository on GitHub.
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/nac.git
   cd nac
   ```
3. Build the binary:
   ```bash
   make build
   ```
4. Run the tests:
   ```bash
   make test
   ```

## 🧪 Testing

We use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) for database integration tests. This ensures our SQL queries are compatible with actual n8n schemas.

To run only the integration tests:
```bash
go test ./internal/db/... -v
```

## 📜 Coding Guidelines

- **Style**: Follow standard Go formatting (`go fmt`).
- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/) (e.g., `feat:`, `fix:`, `chore:`).
- **Documentation**: If you add a feature or change configuration, update the `README.md` and templates.

## 🚀 Pull Request Process

1. Create a new branch for your feature or fix.
2. Ensure all tests pass.
3. Submit a Pull Request against the `master` branch.
4. Provide a clear description of the changes and any relevant issue numbers.

## 🛠 Project Structure

- `cmd/nac`: Entry point for the CLI.
- `internal/cmd`: CLI command logic (Cobra).
- `internal/db`: Database interactions (Direct SQL).
- `internal/credential`: Credential extraction and re-building logic.
- `internal/workflow`: Workflow parsing and re-mapping logic.
- `internal/crypto`: AES-256-CBC implementation (n8n compatible).
- `schema/`: SQL snapshots used for testing and version tracking.
- `docs/`: Historical project plan and troubleshooting guides.

---

Thank you for contributing to Crymfox Labs!
