package credential

import (
	"testing"

	"github.com/crymfox/nac/internal/config"
)

func TestRegistry_ExtractSecrets(t *testing.T) {
	r := NewRegistry(config.Defaults().CredentialTypes)

	data := map[string]any{
		"apiKey":         "sk-123",
		"organizationId": "org-456",
	}

	secrets := r.ExtractSecrets("openAiApi", "my_openai", data)

	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(secrets))
	}

	if val, ok := secrets["MY_OPENAI_API_KEY"]; !ok || val != "sk-123" {
		t.Errorf("Secret value wrong or missing: %v", secrets)
	}
}

func TestRegistry_ExtractSecrets_InstanceOverride(t *testing.T) {
	r := NewRegistry(config.Defaults().CredentialTypes)

	data := map[string]any{
		"name":  "Authorization",
		"value": "Bearer mytoken",
	}

	secrets := r.ExtractSecrets("httpHeaderAuth", "whapi_account", data)

	if len(secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(secrets))
	}

	// Should extract the secret and strip "Bearer " because whapi has bearer_prefix transform
	if val, ok := secrets["WHAPI_API_TOKEN"]; !ok || val != "mytoken" {
		t.Errorf("Secret value wrong or missing: %v", secrets)
	}
}
