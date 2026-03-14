package store

import (
	"database/sql"
	"fmt"

	"github.com/clash-proxyd/proxyd/internal/types"
)

// SettingStore handles setting operations
type SettingStore struct {
	db *DB
}

// NewSettingStore creates a new setting store
func NewSettingStore(db *DB) *SettingStore {
	return &SettingStore{db: db}
}

// Get retrieves a setting by key
func (s *SettingStore) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

// Set sets a setting value
func (s *SettingStore) Set(key, value, description string) error {
	query := `
		INSERT INTO settings (key, value, description, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			description = excluded.description,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.db.Exec(query, key, value, description)
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

// GetAll retrieves all settings
func (s *SettingStore) GetAll() ([]types.Setting, error) {
	rows, err := s.db.Query("SELECT key, value, description, updated_at FROM settings ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	defer rows.Close()

	settings := make([]types.Setting, 0)
	for rows.Next() {
		var setting types.Setting
		err := rows.Scan(&setting.Key, &setting.Value, &setting.Description, &setting.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// GetMap retrieves all settings as a map
func (s *SettingStore) GetMap() (map[string]string, error) {
	settings, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}

	return result, nil
}

// SetBatch sets multiple settings in a transaction
func (s *SettingStore) SetBatch(settings []types.Setting) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin setting transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (key, value, description, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			description = excluded.description,
			updated_at = CURRENT_TIMESTAMP
	`

	for _, setting := range settings {
		if _, err := tx.Exec(query, setting.Key, setting.Value, setting.Description); err != nil {
			return fmt.Errorf("failed to set setting %s: %w", setting.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit setting transaction: %w", err)
	}

	return nil
}

// Delete deletes a setting
func (s *SettingStore) Delete(key string) error {
	result, err := s.db.Exec("DELETE FROM settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}

	return nil
}
