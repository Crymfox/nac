package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crymfox/nac/internal/db"
)

// ExportOptions configures the export process.
type ExportOptions struct {
	Client       *db.Client
	WorkflowsDir string
	IgnoreFields []string
	DryRun       bool
	Verbose      bool
}

// ExportResult summarizes the export process.
type ExportResult struct {
	Updated   int
	Unchanged int
	Removed   int
	Errors    []error
}

// Export fetches all workflows from the DB and writes them to files.
func Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	res := &ExportResult{}

	wfs, err := opts.Client.ListWorkflows(ctx)
	if err != nil {
		return res, fmt.Errorf("listing workflows: %w", err)
	}

	if !opts.DryRun {
		if err := os.MkdirAll(opts.WorkflowsDir, 0o755); err != nil {
			return res, fmt.Errorf("creating workflows dir: %w", err)
		}
	}

	// Track which folders we expect to exist based on the DB
	expectedFolders := make(map[string]bool)

	for _, wf := range wfs {
		if wf.Name == "" {
			if opts.Verbose {
				fmt.Printf("Skipping workflow with empty name (ID: %s)\n", wf.ID)
			}
			continue
		}

		folderName := SanitizeFolderName(wf.Name)
		expectedFolders[folderName] = true

		targetDir := filepath.Join(opts.WorkflowsDir, folderName)
		targetFile := filepath.Join(targetDir, "workflow.json")

		// Convert DB row to a generic map for JSON serialization
		wfMap := workflowToMap(wf)

		// Create normalized version for comparison (strips ignore_fields)
		newNormalized := normalizeForDiff(wfMap, opts.IgnoreFields)

		// Check if file exists and compare
		changed := true
		if existingData, err := os.ReadFile(targetFile); err == nil {
			var existingMap map[string]any
			if err := json.Unmarshal(existingData, &existingMap); err == nil {
				existingNormalized := normalizeForDiff(existingMap, opts.IgnoreFields)
				if mapsEqual(existingNormalized, newNormalized) {
					changed = false
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

		// Write new file
		res.Updated++
		if opts.Verbose {
			fmt.Printf("Updated: %s\n", targetFile)
		}

		if !opts.DryRun {
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("creating dir %s: %w", targetDir, err))
				continue
			}

			// Marshal the FULL workflow map (not just the normalized one)
			outBytes, err := json.MarshalIndent(wfMap, "", "  ")
			if err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("marshaling %s: %w", wf.Name, err))
				continue
			}

			if err := os.WriteFile(targetFile, outBytes, 0o644); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("writing %s: %w", targetFile, err))
				continue
			}
		}
	}

	// Remove stale folders
	entries, err := os.ReadDir(opts.WorkflowsDir)
	if err != nil && !os.IsNotExist(err) {
		return res, fmt.Errorf("reading workflows dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip dotfiles like .gitkeep
		if entry.Name() != "" && entry.Name()[0] == '.' {
			continue
		}

		if !expectedFolders[entry.Name()] {
			res.Removed++
			path := filepath.Join(opts.WorkflowsDir, entry.Name())
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

// workflowToMap converts a db.Workflow into a generic map, matching n8n's JSON export format.
func workflowToMap(wf db.Workflow) map[string]any {
	m := map[string]any{
		"id":         wf.ID,
		"name":       wf.Name,
		"active":     wf.Active,
		"isArchived": wf.IsArchived,
	}

	if wf.VersionID != "" {
		m["versionId"] = wf.VersionID
	}
	if wf.ActiveVersionID != "" {
		m["activeVersionId"] = wf.ActiveVersionID
	}
	if !wf.CreatedAt.IsZero() {
		m["createdAt"] = wf.CreatedAt.Format("2006-01-02T15:04:05.000Z")
	}
	if !wf.UpdatedAt.IsZero() {
		m["updatedAt"] = wf.UpdatedAt.Format("2006-01-02T15:04:05.000Z")
	}

	parseJSONField(wf.Nodes, "nodes", m)
	parseJSONField(wf.Connections, "connections", m)
	parseJSONField(wf.Settings, "settings", m)
	parseJSONField(wf.StaticData, "staticData", m)
	parseJSONField(wf.PinData, "pinData", m)
	parseJSONField(wf.Meta, "meta", m)

	return m
}

func parseJSONField(data []byte, key string, m map[string]any) {
	if len(data) == 0 {
		return
	}
	var val any
	if err := json.Unmarshal(data, &val); err == nil {
		m[key] = val
	}
}

// normalizeForDiff returns a deep copy of the map with ignoreFields removed.
func normalizeForDiff(m map[string]any, ignoreFields []string) map[string]any {
	// Simple deep copy via JSON
	b, _ := json.Marshal(m)
	var copy map[string]any
	json.Unmarshal(b, &copy)

	for _, field := range ignoreFields {
		delete(copy, field)
	}

	// Normalizations
	if _, ok := copy["active"]; !ok {
		copy["active"] = false
	}
	if _, ok := copy["isArchived"]; !ok {
		copy["isArchived"] = false
	}

	return copy
}

// mapsEqual deeply compares two maps.
func mapsEqual(a, b map[string]any) bool {
	jsonA, _ := json.Marshal(a)
	jsonB, _ := json.Marshal(b)
	return string(jsonA) == string(jsonB)
}
