package config

// Build-time variables set via ldflags.
var (
	// Version is the nac CLI version (set by goreleaser or ldflags).
	Version = "0.1.2"

	// Commit is the git commit hash.
	Commit = "none"

	// Date is the build date.
	Date = "unknown"
)

// PinnedN8NVersion is the n8n version that nac's SQL and crypto are validated against.
const PinnedN8NVersion = "2.3.4"
