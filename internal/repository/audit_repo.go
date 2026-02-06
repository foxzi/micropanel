package repository

import (
	"micropanel/internal/database"
	"micropanel/internal/models"
)

type AuditRepository struct {
	db *database.DB
}

func NewAuditRepository(db *database.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(log *models.AuditLog) error {
	result, err := r.db.Exec(
		`INSERT INTO audit_log (user_id, action, entity_type, entity_id, details, ip)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		log.UserID, log.Action, log.EntityType, log.EntityID, log.Details, log.IP,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = id
	return nil
}

func (r *AuditRepository) List(limit, offset int) ([]*models.AuditLog, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, action, entity_type, entity_id, details, ip, created_at
		 FROM audit_log
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.EntityType,
			&log.EntityID, &log.Details, &log.IP, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *AuditRepository) ListByUser(userID int64, limit, offset int) ([]*models.AuditLog, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, action, entity_type, entity_id, details, ip, created_at
		 FROM audit_log
		 WHERE user_id = ?
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.EntityType,
			&log.EntityID, &log.Details, &log.IP, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *AuditRepository) ListByEntity(entityType string, entityID int64, limit, offset int) ([]*models.AuditLog, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, action, entity_type, entity_id, details, ip, created_at
		 FROM audit_log
		 WHERE entity_type = ? AND entity_id = ?
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		entityType, entityID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Action, &log.EntityType,
			&log.EntityID, &log.Details, &log.IP, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *AuditRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&count)
	return count, err
}

func (r *AuditRepository) CountByUser(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

func (r *AuditRepository) DeleteOlderThan(days int) (int64, error) {
	result, err := r.db.Exec(
		`DELETE FROM audit_log WHERE created_at < datetime('now', '-' || ? || ' days')`,
		days,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
