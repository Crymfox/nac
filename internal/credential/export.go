package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crymfox/nac/internal/config"
	"github.com/crymfox/nac/internal/crypto"
	"github.com/crymfox/nac/internal/db"
	"github.com/crymfox/nac/internal/workflow"
)

// ExportOptions configures the credential export process.
type ExportOptions struct {
	Client         *db.Client
	CredentialsDir string
	Types          map[string]config.CredentialType
	EncryptionKey  string
	DryRun         bool
	Verbose        bool
}

// ExportResult summarizes the export process.
type ExportResult struct {
	Updated   int
	Unchanged int
	Removed   int
	Errors    []error
}

// Export fetches all credentials, decrypts them, replaces secrets with placeholders, and writes JSON files.
func Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	res := &ExportResult{}

	creds, err := opts.Client.ListCredentials(ctx)
	if err != nil {
		return res, fmt.Errorf("listing credentials: %w", err)
	}

	if !opts.DryRun {
		if err := os.MkdirAll(opts.CredentialsDir, 0o755); err != nil {
			return res, fmt.Errorf("creating credentials dir: %w", err)
		}
	}

	registry := NewRegistry(opts.Types)
	expectedFolders := make(map[string]bool)

	for _, cred := range creds {
		if cred.Name == "" {
			continue
		}

		folderName := workflow.SanitizeFolderName(cred.Name)
		expectedFolders[folderName] = true

		targetDir := filepath.Join(opts.CredentialsDir, folderName)
		targetFile := filepath.Join(targetDir, "credential.json")

		// Decrypt data
		decryptedJSON, err := crypto.Decrypt(cred.Data, opts.EncryptionKey)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("decrypting %s: %w", cred.Name, err))
			continue
		}

		var dataMap map[string]any
		if err := json.Unmarshal([]byte(decryptedJSON), &dataMap); err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("parsing decrypted data for %s: %w", cred.Name, err))
			continue
		}

		// Create the placeholder version for writing
		placeholderData := registry.ReplaceSecrets(cred.Type, folderName, dataMap)

		// Check if file exists and compare
		changed := true
		if existingFile, err := os.ReadFile(targetFile); err == nil {
			var existingCred map[string]any
			if err := json.Unmarshal(existingFile, &existingCred); err == nil {
				// First check metadata
				existingName, _ := existingCred["name"].(string)
				existingType, _ := existingCred["type"].(string)
				existingId, _ := existingCred["id"].(string)

				if existingName == cred.Name && existingType == cred.Type && existingId == cred.ID {
					// We serialize what we WOULD write, and compare it
					// directly to what is on disk! Since we deterministically replace secrets with
					// the exact same ENV: placeholder string every time.

					// Let's build what we want to write
					newCredMap := map[string]any{
						"id":   cred.ID,
						"name": cred.Name,
						"type": cred.Type,
					}

					// Let's serialize the placeholder map to a compact JSON string
					placeholderBytes, _ := json.Marshal(placeholderData)
					newCredMap["data"] = string(placeholderBytes)

					newCredBytes, _ := json.MarshalIndent(newCredMap, "", "  ")

					// If the generated JSON exactly matches disk, no change.
					if string(newCredBytes) == string(existingFile) {
						changed = false
					}
				}
			}
		}

		if !changed {
			res.Unchanged++
			if opts.Verbose {
				fmt.Printf("Unchanged: %s\n", targetFile)
			}
			continue
		}

		// Write
		res.Updated++
		if opts.Verbose {
			fmt.Printf("Updated: %s\n", targetFile)
		}

		if !opts.DryRun {
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("creating dir %s: %w", targetDir, err))
				continue
			}

			// Build final export map
			newCredMap := map[string]any{
				"id":   cred.ID,
				"name": cred.Name,
				"type": cred.Type,
			}
			placeholderBytes, _ := json.Marshal(placeholderData)
			newCredMap["data"] = string(placeholderBytes)

			outBytes, err := json.MarshalIndent(newCredMap, "", "  ")
			if err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("marshaling %s: %w", cred.Name, err))
				continue
			}

			if err := os.WriteFile(targetFile, outBytes, 0o644); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("writing %s: %w", targetFile, err))
			}
		}
	}

	// Remove stale folders
	entries, err := os.ReadDir(opts.CredentialsDir)
	if err != nil && !os.IsNotExist(err) {
		return res, fmt.Errorf("reading credentials dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() != "" && entry.Name()[0] == '.' {
			continue
		}
		if !expectedFolders[entry.Name()] {
			res.Removed++
			path := filepath.Join(opts.CredentialsDir, entry.Name())
			if opts.Verbose {
				fmt.Printf("Removed stale folder: %s\n", path)
			}
			if !opts.DryRun {
				if err := os.RemoveAll(path); err != nil {
					res.Errors = append(res.Errors, fmt.Errorf("removing stale dir %s: %w", path, err))
				}
			}
		}
	}

	return res, nil
}
