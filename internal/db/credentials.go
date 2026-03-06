package db

import (
	"context"
	"fmt"
	"time"
)

// Credential represents a row in the credentials_entity table.
type Credential struct {
	ID        string
	Name      string
	Type      string
	Data      string // Encrypted Base64 payload
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListCredentials returns all credentials from the database.
func (c *Client) ListCredentials(ctx context.Context) ([]Credential, error) {
	query := `
		SELECT id, name, type, data, "createdAt", "updatedAt"
		FROM credentials_entity
	`
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying credentials: %w", err)
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var cred Credential
		err := rows.Scan(
			&cred.ID, &cred.Name, &cred.Type, &cred.Data,
			&cred.CreatedAt, &cred.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning credential row: %w", err)
		}
		creds = append(creds, cred)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating credentials: %w", err)
	}
	return creds, nil
}

// GetCredentialNameIdMap returns a map of credential names to their IDs.
func (c *Client) GetCredentialNameIdMap(ctx context.Context) (map[string]string, error) {
	query := `SELECT name, id FROM credentials_entity`
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying credential name-id map: %w", err)
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

// UpsertCredential inserts or updates a credential by ID.
func (c *Client) UpsertCredential(ctx context.Context, cred Credential) error {
	query := `
		INSERT INTO credentials_entity (
			id, name, type, data, "createdAt", "updatedAt"
		) VALUES (
			$1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			data = EXCLUDED.data,
			"updatedAt" = CURRENT_TIMESTAMP
	`

	_, err := c.pool.Exec(ctx, query,
		cred.ID, cred.Name, cred.Type, cred.Data,
	)
	return err
}

// DeleteCredentialsByNames deletes multiple credentials by name.
func (c *Client) DeleteCredentialsByNames(ctx context.Context, names []string) (int64, error) {
	if len(names) == 0 {
		return 0, nil
	}
	query := `DELETE FROM credentials_entity WHERE name = ANY($1)`
	tag, err := c.pool.Exec(ctx, query, names)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// EnsureCredentialOwnership ensures that a credential is linked to a project in shared_credentials.
func (c *Client) EnsureCredentialOwnership(ctx context.Context, credentialsId, projectId string) error {
	query := `
		INSERT INTO shared_credentials ("credentialsId", "projectId", role)
		VALUES ($1, $2, 'credential:owner')
		ON CONFLICT ("credentialsId", "projectId") DO NOTHING
	`
	_, err := c.pool.Exec(ctx, query, credentialsId, projectId)
	return err
}
