package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/crymfox/nac/internal/db"
	"github.com/google/uuid"
)

// ImportOptions configures the import process.
type ImportOptions struct {
	Client        *db.Client
	WorkflowsDir  string
	MirrorDeletes bool
	PublishActive bool
	DryRun        bool
	Verbose       bool
}

// ImportResult summarizes the import process.
type ImportResult struct {
	Imported int
	Deleted  int
	Errors   []error
}

// Import reads workflow JSON files and upserts them into the database.
func Import(ctx context.Context, opts ImportOptions) (*ImportResult, error) {
	res := &ImportResult{}

	// 1. Find all workflow.json files
	var files []string
	err := filepath.Walk(opts.WorkflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "workflow.json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return res, fmt.Errorf("walking workflows dir: %w", err)
	}
	if os.IsNotExist(err) {
		return res, nil
	}

	if len(files) == 0 {
		return res, nil
	}

	// 2. Parse all workflows and build localIdToName map
	var localWorkflows []map[string]any
	localIdToName := make(map[string]string)
	incomingNames := make(map[string]bool)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("reading %s: %w", file, err))
			continue
		}

		var wf map[string]any
		if err := json.Unmarshal(data, &wf); err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("parsing %s: %w", file, err))
			continue
		}

		id, _ := wf["id"].(string)
		name, _ := wf["name"].(string)

		if name == "" {
			res.Errors = append(res.Errors, fmt.Errorf("workflow in %s has no name", file))
			continue
		}

		if id != "" {
			localIdToName[id] = name
		}

		incomingNames[name] = true
		localWorkflows = append(localWorkflows, wf)
	}

	if len(localWorkflows) == 0 {
		return res, nil
	}

	// 3. Fetch remote name-to-id map
	remoteNameToId, err := opts.Client.GetWorkflowNameIdMap(ctx)
	if err != nil {
		return res, fmt.Errorf("fetching remote workflow IDs: %w", err)
	}

	// 3.1 Fetch personal project ID (for ownership)
	// Retry for up to 10 seconds (n8n might be initializing)
	var projectID string
	for i := 0; i < 10; i++ {
		id, err := opts.Client.GetPersonalProjectID(ctx)
		if err == nil && id != "" {
			projectID = id
			break
		}
		if opts.Verbose {
			fmt.Println("Waiting for n8n to initialize default project...")
		}
		time.Sleep(2 * time.Second)
	}

	if projectID == "" {
		res.Errors = append(res.Errors, fmt.Errorf("no personal project found. If this is a fresh install, please visit the n8n UI to complete setup"))
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
				fmt.Printf("Mirror mode: deleting %d workflows missing from repo\n", len(toDelete))
			}
			affected, err := opts.Client.DeleteWorkflowsByNames(ctx, toDelete)
			if err != nil {
				return res, fmt.Errorf("deleting workflows: %w", err)
			}
			res.Deleted += int(affected)
		}
	}

	// 5. Process and Upsert
	for _, wfMap := range localWorkflows {
		name := wfMap["name"].(string)
		if remoteId, exists := remoteNameToId[name]; exists {
			wfMap["id"] = remoteId
		}

		id, _ := wfMap["id"].(string)
		if id == "" {
			res.Errors = append(res.Errors, fmt.Errorf("workflow %q has no ID", name))
			continue
		}

		if nodesInterface, ok := wfMap["nodes"].([]any); ok {
			remappedNodes := RemapExecuteWorkflowReferences(nodesInterface, localIdToName, remoteNameToId)
			wfMap["nodes"] = remappedNodes
		}

		active, _ := wfMap["active"].(bool)
		isArchived, _ := wfMap["isArchived"].(bool)
		versionId, _ := wfMap["versionId"].(string)
		if versionId == "" || versionId == "0" {
			versionId = uuid.New().String()
		}

		wf := db.Workflow{
			ID:         id,
			Name:       name,
			Active:     active,
			IsArchived: isArchived,
			VersionID:  versionId,
		}

		wf.Nodes = encodeJSON(wfMap["nodes"], "[]")
		wf.Connections = encodeJSON(wfMap["connections"], "{}")
		wf.Settings = encodeJSON(wfMap["settings"], "{}")
		if v, ok := wfMap["staticData"]; ok && v != nil {
			wf.StaticData = encodeJSON(v, "")
		}
		if v, ok := wfMap["pinData"]; ok && v != nil {
			wf.PinData = encodeJSON(v, "")
		}
		if v, ok := wfMap["meta"]; ok && v != nil {
			wf.Meta = encodeJSON(v, "")
		}

		if opts.Verbose {
			fmt.Printf("Upserting workflow: %s\n", name)
		}

		if !opts.DryRun {
			if err := opts.Client.UpsertWorkflow(ctx, wf); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("upserting %q: %w", name, err))
				continue
			}

			// Ensure ownership
			if projectID != "" {
				if err := opts.Client.EnsureWorkflowOwnership(ctx, id, projectID); err != nil {
					res.Errors = append(res.Errors, fmt.Errorf("ensuring ownership for %q: %w", name, err))
				}
			}

			if err := opts.Client.EnforceWorkflowState(ctx, name, active, isArchived, opts.PublishActive); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("enforcing state for %q: %w", name, err))
				continue
			}
		}
		res.Imported++
	}

	return res, nil
}

func encodeJSON(val any, fallback string) []byte {
	if val == nil {
		if fallback != "" {
			return []byte(fallback)
		}
		return nil
	}
	b, err := json.Marshal(val)
	if err != nil {
		if fallback != "" {
			return []byte(fallback)
		}
		return nil
	}
	return b
}
