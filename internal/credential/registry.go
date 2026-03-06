package credential

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crymfox/nac/internal/config"
)

// Registry holds the known credential types.
type Registry struct {
	types map[string]config.CredentialType
}

// NewRegistry creates a new registry from configuration.
func NewRegistry(types map[string]config.CredentialType) *Registry {
	if types == nil {
		types = make(map[string]config.CredentialType)
	}
	return &Registry{types: types}
}

// HasType checks if a credential type is known.
func (r *Registry) HasType(name string) bool {
	_, ok := r.types[name]
	return ok
}

// ExtractStructural extracts only non-secret fields for comparison.
func (r *Registry) ExtractStructural(credType string, data map[string]any) map[string]any {
	ct, ok := r.types[credType]
	if !ok {
		// Unknown type: conservative approach, compare everything
		return cloneMap(data)
	}

	result := make(map[string]any)
	for _, f := range ct.Fields {
		if !f.Secret {
			if val, exists := getNested(data, f.Name); exists {
				setNested(result, f.Name, val)
			}
		}
	}
	return result
}

// ReplaceSecrets returns a new map where secret fields are replaced by ENV: placeholders.
func (r *Registry) ReplaceSecrets(credType string, folderName string, data map[string]any) map[string]any {
	ct, ok := r.types[credType]
	if !ok {
		return cloneMap(data)
	}

	result := cloneMap(data)
	envPrefix := strings.ToUpper(folderName)

	for _, f := range ct.Fields {
		if f.Secret {
			placeholder := "ENV:" + envPrefix
			if f.EnvSuffix != "" {
				placeholder += f.EnvSuffix
			} else if f.Env != "" {
				placeholder = "ENV:" + f.Env
			}
			setNested(result, f.Name, placeholder)
		}
	}
	return result
}

// BuildData resolves env vars and constructs the actual data payload.
func (r *Registry) BuildData(credType string, folderName string) (map[string]any, error) {
	ct, ok := r.types[credType]
	if !ok {
		return nil, fmt.Errorf("unknown credential type: %s", credType)
	}

	result := make(map[string]any)
	envPrefix := strings.ToUpper(folderName)

	// Check if this specific folder is an instance override
	var instanceOverrides map[string]string
	var valueTransform string
	if inst, ok := ct.Instances[folderName]; ok {
		instanceOverrides = inst.Overrides
		valueTransform = inst.Overrides["value_transform"]
	}

	for _, f := range ct.Fields {
		// Determine env var name
		envVar := f.Env
		if envVar == "" {
			envVar = envPrefix + f.EnvSuffix
		}

		var val string

		// Check for instance override (e.g. fixed header name)
		if override, ok := instanceOverrides[f.Name]; ok {
			val = override
		} else if overrideEnv, ok := instanceOverrides[f.Name+"_env"]; ok {
			val = os.Getenv(overrideEnv)
			if val == "" && !f.Optional {
				return nil, fmt.Errorf("required env var %s not set (via override)", overrideEnv)
			}
		} else {
			// Normal env var resolution
			val = os.Getenv(envVar)
			if val == "" && !f.Optional && f.Secret {
				return nil, fmt.Errorf("required env var %s not set", envVar)
			}
		}

		// Apply transform if specified
		if f.Secret && valueTransform == "bearer_prefix" {
			if val != "" && !strings.HasPrefix(val, "Bearer ") {
				val = "Bearer " + val
			}
		}

		if val != "" || !f.Optional {
			setNested(result, f.Name, val)
		}
	}

	return result, nil
}

// GetDisplayName returns the display name for a credential folder.
func (r *Registry) GetDisplayName(credType string, folderName string) string {
	ct, ok := r.types[credType]
	if !ok {
		return folderToDisplayName(folderName)
	}

	if inst, ok := ct.Instances[folderName]; ok && inst.DisplayName != "" {
		return inst.DisplayName
	}

	return folderToDisplayName(folderName)
}

// ExtractSecrets returns a map of env var names to their values for secret fields.
func (r *Registry) ExtractSecrets(credType string, folderName string, data map[string]any) map[string]string {
	ct, ok := r.types[credType]
	if !ok {
		return nil
	}

	result := make(map[string]string)
	envPrefix := strings.ToUpper(folderName)

	// Check if this specific folder is an instance override
	var instanceOverrides map[string]string
	if inst, ok := ct.Instances[folderName]; ok {
		instanceOverrides = inst.Overrides
	}

	for _, f := range ct.Fields {
		if !f.Secret {
			continue
		}

		// Determine env var name
		envVar := f.Env
		if envVar == "" {
			envVar = envPrefix + f.EnvSuffix
		}

		// If there is an instance override for the env var name, use that
		if overrideEnv, ok := instanceOverrides[f.Name+"_env"]; ok {
			envVar = overrideEnv
		}

		if val, exists := getNested(data, f.Name); exists {
			if s, ok := val.(string); ok {
				// Strip Bearer prefix if it was added by value_transform
				if inst, ok := ct.Instances[folderName]; ok && inst.Overrides["value_transform"] == "bearer_prefix" {
					s = strings.TrimPrefix(s, "Bearer ")
				}
				result[envVar] = s
			}
		}
	}
	return result
}

// GetOAuth2Config returns the OAuth2 config if the type requires it.
func (r *Registry) GetOAuth2Config(credType string) *config.OAuth2Config {
	ct, ok := r.types[credType]
	if !ok {
		return nil
	}
	return ct.OAuth2
}

// Helper: folder_name_style -> Folder Name Style
func folderToDisplayName(folder string) string {
	parts := strings.Split(folder, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// Map helpers for nested fields like "oauthTokenData.refresh_token"

func getNested(m map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = m

	for i, part := range parts {
		cmap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		val, ok := cmap[part]
		if !ok {
			return nil, false
		}

		if i == len(parts)-1 {
			return val, true
		}
		current = val
	}
	return nil, false
}

func setNested(m map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := m

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			next = make(map[string]any)
			current[part] = next
			current = next
		}
	}
	current[parts[len(parts)-1]] = value
}

func cloneMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var clone map[string]any
	json.Unmarshal(b, &clone)
	return clone
}
