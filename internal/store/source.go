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

// Create creates a new source
func (s *SourceStore) Create(source *types.Source) error {
	query := `
		INSERT INTO sources (name, type, url, path, update_interval, update_cron, enabled, priority, config_override)
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

// GetByID retrieves a source by ID
func (s *SourceStore) GetByID(id int) (*types.Source, error) {
	query := `
		SELECT id, name, type, url, path, update_interval, update_cron,
		       enabled, priority, config_override, created_at, updated_at
		FROM sources WHERE id = ?
	`
	source := &types.Source{}
	err := s.db.QueryRow(query, id).Scan(
		&source.ID, &source.Name, &source.Type, &source.URL, &source.Path,
		&source.UpdateInterval, &source.UpdateCron, &source.Enabled,
		&source.Priority, &source.ConfigOverride, &source.CreatedAt,
		&source.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("source not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	return source, nil
}

// GetByName retrieves a source by name
func (s *SourceStore) GetByName(name string) (*types.Source, error) {
	query := `
		SELECT id, name, type, url, path, update_interval, update_cron,
		       enabled, priority, config_override, created_at, updated_at
		FROM sources WHERE name = ?
	`
	source := &types.Source{}
	err := s.db.QueryRow(query, name).Scan(
		&source.ID, &source.Name, &source.Type, &source.URL, &source.Path,
		&source.UpdateInterval, &source.UpdateCron, &source.Enabled,
		&source.Priority, &source.ConfigOverride, &source.CreatedAt,
		&source.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("source not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	return source, nil
}

// List retrieves all sources
func (s *SourceStore) List() ([]types.Source, error) {
	query := `
		SELECT id, name, type, url, path, update_interval, update_cron,
		       enabled, priority, config_override, created_at, updated_at
		FROM sources ORDER BY priority DESC, created_at ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	sources := make([]types.Source, 0)
	for rows.Next() {
		var source types.Source
		err := rows.Scan(
			&source.ID, &source.Name, &source.Type, &source.URL, &source.Path,
			&source.UpdateInterval, &source.UpdateCron, &source.Enabled,
			&source.Priority, &source.ConfigOverride, &source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// GetEnabled retrieves all enabled sources ordered by priority
func (s *SourceStore) GetEnabled() ([]types.Source, error) {
	query := `
		SELECT id, name, type, url, path, update_interval, update_cron,
		       enabled, priority, config_override, created_at, updated_at
		FROM sources WHERE enabled = 1 ORDER BY priority DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled sources: %w", err)
	}
	defer rows.Close()

	sources := make([]types.Source, 0)
	for rows.Next() {
		var source types.Source
		err := rows.Scan(
			&source.ID, &source.Name, &source.Type, &source.URL, &source.Path,
			&source.UpdateInterval, &source.UpdateCron, &source.Enabled,
			&source.Priority, &source.ConfigOverride, &source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// Update updates a source
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

// UpdateLastFetch updates the last fetch timestamp
func (s *SourceStore) UpdateLastFetch(id int) error {
	_, err := s.db.Exec("UPDATE sources SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to update last fetch: %w", err)
	}
	return nil
}
