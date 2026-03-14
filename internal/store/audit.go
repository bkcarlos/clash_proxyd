package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/clash-proxyd/proxyd/internal/types"
)

// AuditStore handles audit log operations
type AuditStore struct {
	db *DB
}

// NewAuditStore creates a new audit store
func NewAuditStore(db *DB) *AuditStore {
	return &AuditStore{db: db}
}

// Create creates a new audit log entry
func (a *AuditStore) Create(log *types.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user, action, resource, details, ip_address)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := a.db.Exec(
		query,
		log.User, log.Action, log.Resource, log.Details, log.IPAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = int(id)
	log.CreatedAt = time.Now()

	return nil
}

// List retrieves audit logs with pagination
func (a *AuditStore) List(limit, offset int) ([]types.AuditLog, error) {
	query := `
		SELECT id, user, action, resource, details, ip_address, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?
	`
	rows, err := a.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]types.AuditLog, 0)
	for rows.Next() {
		var log types.AuditLog
		var user, resource, details, ipAddress sql.NullString
		err := rows.Scan(
			&log.ID, &user, &log.Action, &resource, &details, &ipAddress, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		if user.Valid {
			log.User = user.String
		}
		if resource.Valid {
			log.Resource = resource.String
		}
		if details.Valid {
			log.Details = details.String
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// ListByUser retrieves audit logs for a specific user
func (a *AuditStore) ListByUser(user string, limit, offset int) ([]types.AuditLog, error) {
	query := `
		SELECT id, user, action, resource, details, ip_address, created_at
		FROM audit_logs WHERE user = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`
	rows, err := a.db.Query(query, user, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs by user: %w", err)
	}
	defer rows.Close()

	logs := make([]types.AuditLog, 0)
	for rows.Next() {
		var log types.AuditLog
		var user, resource, details, ipAddress sql.NullString
		err := rows.Scan(
			&log.ID, &user, &log.Action, &resource, &details, &ipAddress, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		if user.Valid {
			log.User = user.String
		}
		if resource.Valid {
			log.Resource = resource.String
		}
		if details.Valid {
			log.Details = details.String
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// DeleteOld deletes audit logs older than specified days
func (a *AuditStore) DeleteOld(days int) (int64, error) {
	query := `
		DELETE FROM audit_logs
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`
	result, err := a.db.Exec(query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rows, nil
}

// Count returns the total count of audit logs
func (a *AuditStore) Count() (int64, error) {
	var count int64
	err := a.db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return count, nil
}
