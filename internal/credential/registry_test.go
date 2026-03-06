package credential

import (
	"os"
	"testing"

	"github.com/crymfox/nac/internal/config"
)

func TestRegistry_ExtractStructural(t *testing.T) {
	r := NewRegistry(config.Defaults().CredentialTypes)

	data := map[string]any{
		"apiKey":         "sk-123",
		"organizationId": "org-456",
		"url":            "https://api.example.com",
	}

	structural := r.ExtractStructural("openAiApi", data)

	if _, exists := structural["apiKey"]; exists {
		t.Error("apiKey should be stripped (secret)")
	}

	if val, ok := structural["organizationId"].(string); !ok || val != "org-456" {
		t.Errorf("organizationId missing or wrong: %v", structural["organizationId"])
	}

	if val, ok := structural["url"].(string); !ok || val != "https://api.example.com" {
		t.Errorf("url missing or wrong: %v", structural["url"])
	}
}

func TestRegistry_ReplaceSecrets(t *testing.T) {
	r := NewRegistry(config.Defaults().CredentialTypes)

	data := map[string]any{
		"apiKey":         "sk-123",
		"organizationId": "org-456",
	}

	replaced := r.ReplaceSecrets("openAiApi", "my_openai", data)

	if val, ok := replaced["apiKey"].(string); !ok || val != "ENV:MY_OPENAI_API_KEY" {
		t.Errorf("apiKey not replaced correctly: %v", replaced["apiKey"])
	}

	if val, ok := replaced["organizationId"].(string); !ok || val != "org-456" {
		t.Errorf("organizationId should remain unchanged: %v", replaced["organizationId"])
	}
}

func TestRegistry_BuildData(t *testing.T) {
	r := NewRegistry(config.Defaults().CredentialTypes)

	os.Setenv("TEST_OPENAI_API_KEY", "sk-build-123")
	os.Setenv("TEST_OPENAI_ORGANIZATION_ID", "org-build")
	defer os.Unsetenv("TEST_OPENAI_API_KEY")
	defer os.Unsetenv("TEST_OPENAI_ORGANIZATION_ID")

	data, err := r.BuildData("openAiApi", "test_openai")
	if err != nil {
		t.Fatalf("BuildData failed: %v", err)
	}

	if data["apiKey"] != "sk-build-123" {
		t.Errorf("apiKey wrong: %v", data["apiKey"])
	}
	if data["organizationId"] != "org-build" {
		t.Errorf("organizationId wrong: %v", data["organizationId"])
	}
}

func TestRegistry_BuildData_InstanceOverride(t *testing.T) {
	r := NewRegistry(config.Defaults().CredentialTypes)

	os.Setenv("SUPADATA_API_KEY", "sd-123")
	defer os.Unsetenv("SUPADATA_API_KEY")

	data, err := r.BuildData("httpHeaderAuth", "supadata_account")
	if err != nil {
		t.Fatalf("BuildData failed: %v", err)
	}

	if data["name"] != "x-api-key" {
		t.Errorf("name wrong: %v", data["name"])
	}
	if data["value"] != "sd-123" {
		t.Errorf("value wrong: %v", data["value"])
	}
}

func TestNestedMapHelpers(t *testing.T) {
	m := make(map[string]any)

	setNested(m, "a.b.c", "hello")

	if val, ok := getNested(m, "a.b.c"); !ok || val != "hello" {
		t.Errorf("getNested failed: %v", val)
	}

	// Overwrite
	setNested(m, "a.b.c", "world")
	if val, ok := getNested(m, "a.b.c"); !ok || val != "world" {
		t.Errorf("getNested failed: %v", val)
	}

	if _, ok := getNested(m, "a.x"); ok {
		t.Error("getNested should return false for nonexistent")
	}
}
