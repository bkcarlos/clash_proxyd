package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
)

// RuntimeStore handles runtime state operations
type RuntimeStore struct {
	db *DB
}

// NewRuntimeStore creates a new runtime store
func NewRuntimeStore(db *DB) *RuntimeStore {
	return &RuntimeStore{db: db}
}

// Get retrieves the current runtime state
func (r *RuntimeStore) Get() (*types.Runtime, error) {
	query := `
		SELECT id, pid, port, config_path, status, uptime, memory_usage, last_check, updated_at
		FROM runtime ORDER BY id DESC LIMIT 1
	`
	runtime := &types.Runtime{}
	err := r.db.QueryRow(query).Scan(
		&runtime.ID, &runtime.PID, &runtime.Port, &runtime.ConfigPath,
		&runtime.Status, &runtime.Uptime, &runtime.MemoryUsage,
		&runtime.LastCheck, &runtime.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("runtime state not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime: %w", err)
	}
	return runtime, nil
}

// Create creates a new runtime state
func (r *RuntimeStore) Create(runtime *types.Runtime) error {
	query := `
		INSERT INTO runtime (pid, port, config_path, status, uptime, memory_usage)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(
		query,
		runtime.PID, runtime.Port, runtime.ConfigPath,
		runtime.Status, runtime.Uptime, runtime.MemoryUsage,
	)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	runtime.ID = int(id)
	runtime.LastCheck = time.Now()
	runtime.UpdatedAt = time.Now()

	return nil
}

// Update updates the runtime state
func (r *RuntimeStore) Update(runtime *types.Runtime) error {
	query := `
		UPDATE runtime
		SET pid = ?, port = ?, config_path = ?, status = ?,
		    uptime = ?, memory_usage = ?, last_check = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	result, err := r.db.Exec(
		query,
		runtime.PID, runtime.Port, runtime.ConfigPath, runtime.Status,
		runtime.Uptime, runtime.MemoryUsage, runtime.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update runtime: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("runtime not found")
	}

	return nil
}

// UpdateStatus updates only the status field
func (r *RuntimeStore) UpdateStatus(id int, status string) error {
	query := `
		UPDATE runtime
		SET status = ?, last_check = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

// UpdatePID updates the process ID
func (r *RuntimeStore) UpdatePID(id int, pid int) error {
	query := `
		UPDATE runtime SET pid = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`
	_, err := r.db.Exec(query, pid, id)
	if err != nil {
		return fmt.Errorf("failed to update PID: %w", err)
	}
	return nil
}

// GetAllRunning returns all rows with status='running' and a valid PID.
// Used by the startup cleanup to kill stale mihomo processes.
func (r *RuntimeStore) GetAllRunning() ([]*types.Runtime, error) {
	query := `
		SELECT id, pid, port, config_path, status, uptime, memory_usage, last_check, updated_at
		FROM runtime WHERE status = 'running' AND pid > 0
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query running runtimes: %w", err)
	}
	defer rows.Close()

	var results []*types.Runtime
	for rows.Next() {
		rt := &types.Runtime{}
		if err := rows.Scan(&rt.ID, &rt.PID, &rt.Port, &rt.ConfigPath,
			&rt.Status, &rt.Uptime, &rt.MemoryUsage, &rt.LastCheck, &rt.UpdatedAt); err != nil {
			continue
		}
		results = append(results, rt)
	}
	return results, nil
}

// PurgeStaleRows deletes all rows except the one with the highest id.
// Keeps the runtime table from growing unboundedly across restarts.
func (r *RuntimeStore) PurgeStaleRows() error {
	_, err := r.db.Exec(`DELETE FROM runtime WHERE id < (SELECT MAX(id) FROM runtime)`)
	return err
}

// UpdateStats updates uptime and memory usage
func (r *RuntimeStore) UpdateStats(id int, uptime, memoryUsage int) error {
	query := `
		UPDATE runtime
		SET uptime = ?, memory_usage = ?, last_check = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Exec(query, uptime, memoryUsage, id)
	if err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}
	return nil
}
