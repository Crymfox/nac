package db

import (
	"context"
	"fmt"
	"time"
)

// Workflow represents a row in the workflow_entity table.
type Workflow struct {
	ID              string
	Name            string
	Active          bool
	IsArchived      bool
	Nodes           []byte // JSON
	Connections     []byte // JSON
	Settings        []byte // JSON
	StaticData      []byte // JSON
	PinData         []byte // JSON
	Meta            []byte // JSON
	VersionID       string
	ActiveVersionID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ListWorkflows returns all workflows from the database.
func (c *Client) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	query := `
		SELECT
			id, name, active, "isArchived", nodes, connections, settings,
			"staticData", "pinData", meta, "versionId", "activeVersionId",
			"createdAt", "updatedAt"
		FROM workflow_entity
	`
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying workflows: %w", err)
	}
	defer rows.Close()

	var wfs []Workflow
	for rows.Next() {
		var wf Workflow
		var staticData, pinData, meta, versionId, activeVersionId *string
		var nodes, connections, settings []byte

		err := rows.Scan(
			&wf.ID, &wf.Name, &wf.Active, &wf.IsArchived,
			&nodes, &connections, &settings,
			&staticData, &pinData, &meta, &versionId, &activeVersionId,
			&wf.CreatedAt, &wf.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning workflow row: %w", err)
		}

		wf.Nodes = nodes
		wf.Connections = connections
		wf.Settings = settings
		if staticData != nil {
			wf.StaticData = []byte(*staticData)
		}
		if pinData != nil {
			wf.PinData = []byte(*pinData)
		}
		if meta != nil {
			wf.Meta = []byte(*meta)
		}
		if versionId != nil {
			wf.VersionID = *versionId
		}
		if activeVersionId != nil {
			wf.ActiveVersionID = *activeVersionId
		}

		wfs = append(wfs, wf)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating workflows: %w", err)
	}

	return wfs, nil
}

// GetWorkflowNameIdMap returns a map of workflow names to their IDs.
func (c *Client) GetWorkflowNameIdMap(ctx context.Context) (map[string]string, error) {
	query := `SELECT name, id FROM workflow_entity`
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying workflow name-id map: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var name, id string
		if err := rows.Scan(&name, &id); err != nil {
			return nil, err
		}
		m[name] = id
	}
	return m, nil
}

// UpsertWorkflow inserts or updates a workflow by ID.
func (c *Client) UpsertWorkflow(ctx context.Context, wf Workflow) error {
	query := `
		INSERT INTO workflow_entity (
			id, name, active, "isArchived", nodes, connections, settings,
			"staticData", "pinData", meta, "versionId", "createdAt", "updatedAt"
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			active = EXCLUDED.active,
			"isArchived" = EXCLUDED."isArchived",
			nodes = EXCLUDED.nodes,
			connections = EXCLUDED.connections,
			settings = EXCLUDED.settings,
			"staticData" = EXCLUDED."staticData",
			"pinData" = EXCLUDED."pinData",
			meta = EXCLUDED.meta,
			"versionId" = EXCLUDED."versionId",
			"updatedAt" = CURRENT_TIMESTAMP
	`
	var staticData, pinData, meta *string
	if len(wf.StaticData) > 0 {
		s := string(wf.StaticData)
		staticData = &s
	}
	if len(wf.PinData) > 0 {
		s := string(wf.PinData)
		pinData = &s
	}
	if len(wf.Meta) > 0 {
		s := string(wf.Meta)
		meta = &s
	}

	_, err := c.pool.Exec(ctx, query,
		wf.ID, wf.Name, wf.Active, wf.IsArchived,
		string(wf.Nodes), string(wf.Connections), string(wf.Settings),
		staticData, pinData, meta, wf.VersionID,
	)
	return err
}

// DeleteWorkflowsByNames deletes multiple workflows by name.
func (c *Client) DeleteWorkflowsByNames(ctx context.Context, names []string) (int64, error) {
	if len(names) == 0 {
		return 0, nil
	}
	query := `DELETE FROM workflow_entity WHERE name = ANY($1)`
	tag, err := c.pool.Exec(ctx, query, names)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// EnforceWorkflowState sets the active and isArchived flags for a workflow by name.
// If publishActive is true, it also sets activeVersionId = versionId when active is true.
func (c *Client) EnforceWorkflowState(ctx context.Context, name string, active, isArchived, publishActive bool) error {
	var query string
	if publishActive && active {
		query = `UPDATE workflow_entity SET active = $1, "isArchived" = $2, "activeVersionId" = "versionId" WHERE name = $3`
	} else {
		query = `UPDATE workflow_entity SET active = $1, "isArchived" = $2 WHERE name = $3`
	}
	_, err := c.pool.Exec(ctx, query, active, isArchived, name)
	return err
}

// GetActiveWorkflowIDs returns the IDs of all workflows that are active and not archived.
func (c *Client) GetActiveWorkflowIDs(ctx context.Context) ([]string, error) {
	query := `SELECT id FROM workflow_entity WHERE active = true AND "isArchived" = false`
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
