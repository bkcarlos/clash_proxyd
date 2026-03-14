package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
)

// RevisionStore handles revision operations
type RevisionStore struct {
	db *DB
}

// NewRevisionStore creates a new revision store
func NewRevisionStore(db *DB) *RevisionStore {
	return &RevisionStore{db: db}
}

// Create creates a new revision
func (r *RevisionStore) Create(revision *types.Revision) error {
	query := `
		INSERT INTO revisions (version, content, source_hash, created_by)
		VALUES (?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, revision.Version, revision.Content, revision.SourceHash, revision.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to create revision: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	revision.ID = int(id)
	revision.CreatedAt = time.Now()

	return nil
}

// GetByID retrieves a revision by ID
func (r *RevisionStore) GetByID(id int) (*types.Revision, error) {
	query := `
		SELECT id, version, content, source_hash, created_by, created_at
		FROM revisions WHERE id = ?
	`
	revision := &types.Revision{}
	err := r.db.QueryRow(query, id).Scan(
		&revision.ID, &revision.Version, &revision.Content,
		&revision.SourceHash, &revision.CreatedBy, &revision.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revision not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get revision: %w", err)
	}
	return revision, nil
}

// GetByVersion retrieves a revision by version
func (r *RevisionStore) GetByVersion(version string) (*types.Revision, error) {
	query := `
		SELECT id, version, content, source_hash, created_by, created_at
		FROM revisions WHERE version = ?
	`
	revision := &types.Revision{}
	err := r.db.QueryRow(query, version).Scan(
		&revision.ID, &revision.Version, &revision.Content,
		&revision.SourceHash, &revision.CreatedBy, &revision.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("revision not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get revision: %w", err)
	}
	return revision, nil
}

// List retrieves all revisions
func (r *RevisionStore) List(limit int) ([]types.Revision, error) {
	query := `
		SELECT id, version, content, source_hash, created_by, created_at
		FROM revisions ORDER BY created_at DESC
	`
	if limit > 0 {
		query += " LIMIT ?"
	}

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list revisions: %w", err)
	}
	defer rows.Close()

	revisions := make([]types.Revision, 0)
	for rows.Next() {
		var revision types.Revision
		err := rows.Scan(
			&revision.ID, &revision.Version, &revision.Content,
			&revision.SourceHash, &revision.CreatedBy, &revision.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revision: %w", err)
		}
		revisions = append(revisions, revision)
	}

	return revisions, nil
}

// GetLatest retrieves the latest revision
func (r *RevisionStore) GetLatest() (*types.Revision, error) {
	query := `
		SELECT id, version, content, source_hash, created_by, created_at
		FROM revisions ORDER BY created_at DESC LIMIT 1
	`
	revision := &types.Revision{}
	err := r.db.QueryRow(query).Scan(
		&revision.ID, &revision.Version, &revision.Content,
		&revision.SourceHash, &revision.CreatedBy, &revision.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest revision: %w", err)
	}
	return revision, nil
}

// Delete deletes a revision
func (r *RevisionStore) Delete(id int) error {
	result, err := r.db.Exec("DELETE FROM revisions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete revision: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("revision not found")
	}

	return nil
}

// DeleteOld deletes revisions older than specified days
func (r *RevisionStore) DeleteOld(days int) (int64, error) {
	query := `
		DELETE FROM revisions
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`
	result, err := r.db.Exec(query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old revisions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rows, nil
}
