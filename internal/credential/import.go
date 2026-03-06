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
)

// ImportOptions configures the credential import process.
type ImportOptions struct {
	Client         *db.Client
	CredentialsDir string
	Types          map[string]config.CredentialType
	EncryptionKey  string
	OldKeys        []string // For key migration
	MirrorDeletes  bool
	DryRun         bool
	Verbose        bool
}

// ImportResult summarizes the import process.
type ImportResult struct {
	Imported int
	Deleted  int
	Migrated int
	Errors   []error
}

// Import reads credential JSON files, builds data from env vars, encrypts, and upserts to DB.
func Import(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	res := &ImportResult{}
	registry := NewRegistry(opts.Types)

	// 1. Find all credential.json files
	var files []string
	err := filepath.Walk(opts.CredentialsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "credential.json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return res, fmt.Errorf("walking credentials dir: %w", err)
	}
	if os.IsNotExist(err) {
		return res, nil
	}

	if len(files) == 0 {
		return res, nil
	}

	// 2. Parse all credentials
	var localCreds []db.Credential
	incomingNames := make(map[string]bool)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("reading %s: %w", file, err))
			continue
		}

		var credMap map[string]any
		if err := json.Unmarshal(data, &credMap); err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("parsing %s: %w", file, err))
			continue
		}

		id, _ := credMap["id"].(string)
		credType, _ := credMap["type"].(string)
		name, _ := credMap["name"].(string)

		folderName := filepath.Base(filepath.Dir(file))

		// Priority: Use name from JSON, Fallback: Derive from folder
		displayName := name
		if displayName == "" {
			displayName = registry.GetDisplayName(credType, folderName)
		}

		if !registry.HasType(credType) {
			res.Errors = append(res.Errors, fmt.Errorf("unknown credential type %q in %s", credType, file))
			continue
		}

		// Build data from env vars
		builtData, err := registry.BuildData(credType, folderName)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("building data for %s: %w", displayName, err))
			continue
		}

		// Handle OAuth2 Token Refresh if needed
		if oauthCfg := registry.GetOAuth2Config(credType); oauthCfg != nil && oauthCfg.AutoRefresh {
			if err := performOAuthRefresh(builtData, oauthCfg); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("oauth2 refresh for %s failed: %w", displayName, err))
				continue
			}
			if opts.Verbose {
				fmt.Printf("Refreshed OAuth2 token for %s\n", displayName)
			}
		}

		// Encrypt
		builtBytes, _ := json.Marshal(builtData)
		encryptedData, err := crypto.Encrypt(string(builtBytes), opts.EncryptionKey)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("encrypting %s: %w", displayName, err))
			continue
		}

		cred := db.Credential{
			ID:   id,
			Name: displayName,
			Type: credType,
			Data: encryptedData,
		}

		incomingNames[displayName] = true
		localCreds = append(localCreds, cred)
	}

	if len(localCreds) == 0 {
		return res, nil
	}

	// 3. Fetch remote name-to-id map
	remoteNameToId, err := opts.Client.GetCredentialNameIdMap(ctx)
	if err != nil {
		return res, fmt.Errorf("fetching remote credential IDs: %w", err)
	}

	// 4. Mirror deletes
	if opts.MirrorDeletes && !opts.DryRun {
		var toDelete []string
		for remoteName := range remoteNameToId {
			if !incomingNames[remoteName] {
				toDelete = append(toDelete, remoteName)
			}
		}

		if len(toDelete) > 0 {
			if opts.Verbose {
				fmt.Printf("Mirror mode: deleting %d credentials missing from repo\n", len(toDelete))
			}
			affected, err := opts.Client.DeleteCredentialsByNames(ctx, toDelete)
			if err != nil {
				return res, fmt.Errorf("deleting credentials: %w", err)
			}
			res.Deleted += int(affected)
		}
	}

	// 5. Process and Upsert
	for _, cred := range localCreds {
		if remoteId, exists := remoteNameToId[cred.Name]; exists {
			cred.ID = remoteId
		}

		if cred.ID == "" {
			res.Errors = append(res.Errors, fmt.Errorf("credential %q has no ID", cred.Name))
			continue
		}

		if opts.Verbose {
			fmt.Printf("Upserting credential: %s\n", cred.Name)
		}

		if !opts.DryRun {
			if err := opts.Client.UpsertCredential(ctx, cred); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("upserting %q: %w", cred.Name, err))
				continue
			}
		}
		res.Imported++
	}

	// 6. Handle Encryption Key Migration (for credentials not in our local list)
	// We only migrate credentials already in the DB that we didn't just update.
	if len(opts.OldKeys) > 0 && !opts.DryRun {
		allCreds, err := opts.Client.ListCredentials(ctx)
		if err != nil {
			return res, fmt.Errorf("listing credentials for migration: %w", err)
		}

		for _, cred := range allCreds {
			// Skip if we just imported it (it's already using the new key)
			if incomingNames[cred.Name] {
				continue
			}

			// Try to decrypt with new key to see if it's already migrated
			if _, err := crypto.Decrypt(cred.Data, opts.EncryptionKey); err == nil {
				continue
			}

			// Try old keys
			var decryptedData string
			var decrypted bool
			for _, oldKey := range opts.OldKeys {
				if plain, err := crypto.Decrypt(cred.Data, oldKey); err == nil {
					decryptedData = plain
					decrypted = true
					break
				}
			}

			if decrypted {
				// Re-encrypt with new key
				newEnc, err := crypto.Encrypt(decryptedData, opts.EncryptionKey)
				if err != nil {
					res.Errors = append(res.Errors, fmt.Errorf("re-encrypting %s: %w", cred.Name, err))
					continue
				}

				cred.Data = newEnc
				if err := opts.Client.UpsertCredential(ctx, cred); err != nil {
					res.Errors = append(res.Errors, fmt.Errorf("upserting migrated %s: %w", cred.Name, err))
					continue
				}
				res.Migrated++
				if opts.Verbose {
					fmt.Printf("Migrated encryption key for: %s\n", cred.Name)
				}
			} else {
				res.Errors = append(res.Errors, fmt.Errorf("could not decrypt %s with any provided key", cred.Name))
			}
		}
	}

	return res, nil
}

func performOAuthRefresh(data map[string]any, cfg *config.OAuth2Config) error {
	clientId, _ := data["clientId"].(string)
	clientSecret, _ := data["clientSecret"].(string)

	// Extract refresh token from nested oauthTokenData
	var refreshToken string
	if tokenData, ok := data["oauthTokenData"].(map[string]any); ok {
		refreshToken, _ = tokenData["refresh_token"].(string)
	}

	if clientId == "" || clientSecret == "" || refreshToken == "" {
		return fmt.Errorf("missing oauth2 parameters")
	}

	result, err := RefreshOAuth2Token(cfg.TokenURL, clientId, clientSecret, refreshToken)
	if err != nil {
		return err
	}

	// Update token data
	accessToken, _ := result["access_token"].(string)
	expiresIn, _ := result["expires_in"].(float64)
	scope, _ := result["scope"].(string)
	if scope == "" {
		scope = cfg.ScopeDefault
	}

	if accessToken == "" {
		return fmt.Errorf("no access_token in refresh response")
	}

	// Make sure oauthTokenData exists
	tokenData, ok := data["oauthTokenData"].(map[string]any)
	if !ok {
		tokenData = make(map[string]any)
		data["oauthTokenData"] = tokenData
	}

	tokenData["access_token"] = accessToken
	tokenData["expires_in"] = expiresIn
	if newRefresh, ok := result["refresh_token"].(string); ok && newRefresh != "" {
		tokenData["refresh_token"] = newRefresh
	} else {
		tokenData["refresh_token"] = refreshToken // Keep old if not provided
	}
	tokenData["scope"] = scope
	tokenData["token_type"] = "Bearer"

	return nil
}
