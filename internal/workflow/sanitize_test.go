package workflow

import (
	"testing"
)

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Workflow", "my_workflow"},
		{"API - Test v2.0", "api_test_v2_0"},
		{"Test (part 1)", "test_(part_1)"},
		{"already_clean", "already_clean"},
		{"___leading___trailing___", "leading_trailing"},
		{"multiple!@#$%^&*()chars", "multiple_()chars"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeFolderName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFolderName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemapExecuteWorkflowReferences(t *testing.T) {
	nodes := []any{
		map[string]any{
			"type": "n8n-nodes-base.executeWorkflow",
			"parameters": map[string]any{
				"workflowId": map[string]any{
					"value":            "old-local-id",
					"cachedResultName": "My Target Workflow",
				},
			},
		},
		map[string]any{
			"type": "n8n-nodes-base.httpRequest",
			"parameters": map[string]any{
				"url": "https://example.com",
			},
		},
	}

	localIdToName := map[string]string{
		"old-local-id": "My Target Workflow",
	}

	remoteNameToId := map[string]string{
		"My Target Workflow": "new-remote-id",
	}

	remapped := RemapExecuteWorkflowReferences(nodes, localIdToName, remoteNameToId)

	// Check the executeWorkflow node
	execNode := remapped[0].(map[string]any)
	params := execNode["parameters"].(map[string]any)
	wfIdMap := params["workflowId"].(map[string]any)

	if wfIdMap["value"] != "new-remote-id" {
		t.Errorf("Expected value to be 'new-remote-id', got %v", wfIdMap["value"])
	}
	if wfIdMap["cachedResultUrl"] != "/workflow/new-remote-id" {
		t.Errorf("Expected cachedResultUrl to be '/workflow/new-remote-id', got %v", wfIdMap["cachedResultUrl"])
	}

	// Check that the httpRequest node was untouched
	httpNode := remapped[1].(map[string]any)
	if httpNode["type"] != "n8n-nodes-base.httpRequest" {
		t.Errorf("httpRequest node modified")
	}
}

func TestNormalizeForDiff(t *testing.T) {
	input := map[string]any{
		"id":        "123",
		"name":      "Test",
		"createdAt": "2023-01-01",
		"updatedAt": "2023-01-02",
		"versionId": "v1",
	}

	ignoreFields := []string{"createdAt", "updatedAt", "versionId"}

	normalized := normalizeForDiff(input, ignoreFields)

	if _, exists := normalized["createdAt"]; exists {
		t.Error("createdAt should be ignored")
	}
	if _, exists := normalized["updatedAt"]; exists {
		t.Error("updatedAt should be ignored")
	}
	if _, exists := normalized["versionId"]; exists {
		t.Error("versionId should be ignored")
	}

	if normalized["id"] != "123" {
		t.Errorf("id should be kept, got %v", normalized["id"])
	}

	// Default active/isArchived insertion
	if normalized["active"] != false {
		t.Errorf("active should default to false")
	}
}
