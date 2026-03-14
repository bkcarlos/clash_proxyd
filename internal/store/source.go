package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
)

// SourceStore handles source operations
type SourceStore struct {
	db *DB
}

// NewSourceStore creates a new source store
func NewSourceStore(db *DB) *SourceStore {
	return &SourceStore{db: db}
}

// scanSource scans a source row including the cached content fields.
func scanSource(scan func(dest ...any) error) (types.Source, error) {
	var src types.Source
	var content sql.NullString
	var contentSize sql.NullInt64
	var lastFetch sql.NullTime
	err := scan(
		&src.ID, &src.Name, &src.Type, &src.URL, &src.Path,
		&src.UpdateInterval, &src.UpdateCron, &src.Enabled,
		&src.Priority, &src.ConfigOverride,
		&content, &contentSize, &lastFetch,
		&src.CreatedAt, &src.UpdatedAt,
	)
	if err != nil {
		return src, err
	}
	if content.Valid {
		src.Content = content.String
	}
	if contentSize.Valid {
		src.ContentSize = int(contentSize.Int64)
	}
	if lastFetch.Valid {
		t := lastFetch.Time
		src.LastFetch = &t
	}
	return src, nil
}

const sourceColumns = `id, name, type, url, path, update_interval, update_cron,
       enabled, priority, config_override, content, content_size, last_fetch,
       created_at, updated_at`

// Create creates a new source
func (s *SourceStore) Create(source *types.Source) error {
	query := `
		INSERT INTO sources (name, type, url, path, update_interval, update_cron,
		                     enabled, priority, config_override)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(
		query,
		source.Name, source.Type, source.URL, source.Path,
		source.UpdateInterval, source.UpdateCron, source.Enabled,
		source.Priority, source.ConfigOverride,
	)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	source.ID = int(id)
	source.CreatedAt = time.Now()
	source.UpdatedAt = time.Now()

	return nil
}

// UpdateContent saves fetched subscription content for a source.
func (s *SourceStore) UpdateContent(id int, content []byte) error {
	now := time.Now()
	_, err := s.db.Exec(
		`UPDATE sources SET content = ?, content_size = ?, last_fetch = ?, updated_at = ? WHERE id = ?`,
		string(content), len(content), now, now, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update source content: %w", err)
	}
	return nil
}

// GetByID retrieves a source by ID
func (s *SourceStore) GetByID(id int) (*types.Source, error) {
	query := `SELECT ` + sourceColumns + ` FROM sources WHERE id = ?`
	src, err := scanSource(s.db.QueryRow(query, id).Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("source not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	return &src, nil
}

// GetByName retrieves a source by name
func (s *SourceStore) GetByName(name string) (*types.Source, error) {
	query := `SELECT ` + sourceColumns + ` FROM sources WHERE name = ?`
	src, err := scanSource(s.db.QueryRow(query, name).Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("source not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	return &src, nil
}

// List retrieves all sources (content field omitted for list performance)
func (s *SourceStore) List() ([]types.Source, error) {
	query := `SELECT ` + sourceColumns + ` FROM sources ORDER BY priority DESC, created_at ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	sources := make([]types.Source, 0)
	for rows.Next() {
		src, err := scanSource(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		src.Content = "" // omit raw content from list responses
		sources = append(sources, src)
	}

	return sources, nil
}

// GetEnabled retrieves all enabled sources ordered by priority
func (s *SourceStore) GetEnabled() ([]types.Source, error) {
	query := `SELECT ` + sourceColumns + ` FROM sources WHERE enabled = 1 ORDER BY priority DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled sources: %w", err)
	}
	defer rows.Close()

	sources := make([]types.Source, 0)
	for rows.Next() {
		src, err := scanSource(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources = append(sources, src)
	}

	return sources, nil
}

// Update updates source metadata (does not touch cached content)
func (s *SourceStore) Update(source *types.Source) error {
	query := `
		UPDATE sources
		SET name = ?, type = ?, url = ?, path = ?, update_interval = ?,
		    update_cron = ?, enabled = ?, priority = ?, config_override = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	result, err := s.db.Exec(
		query,
		source.Name, source.Type, source.URL, source.Path,
		source.UpdateInterval, source.UpdateCron, source.Enabled,
		source.Priority, source.ConfigOverride, source.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("source not found")
	}

	return nil
}

// Delete deletes a source
func (s *SourceStore) Delete(id int) error {
	result, err := s.db.Exec("DELETE FROM sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("source not found")
	}

	return nil
}

// UpdateLastFetch updates the last fetch timestamp (legacy, prefer UpdateContent)
func (s *SourceStore) UpdateLastFetch(id int) error {
	_, err := s.db.Exec(
		"UPDATE sources SET last_fetch = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id,
	)
	if err != nil {
		return fmt.Errorf("failed to update last fetch: %w", err)
	}
	return nil
}
