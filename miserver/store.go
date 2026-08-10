package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type EnvRecord struct {
	ID               int64
	DeviceCode       string
	DeviceID         string
	Type             string
	SerialBackupName string
	AndroidID        string
	Key              string
	UsageCount       int
	MaxUsage         int
	Frozen           bool
	ConsumedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

type EnvFilter struct {
	Type             string
	DeviceCode       string
	DeviceID         string
	SerialBackupName string
	AndroidID        string
	Key              string
	Frozen           *int
	Limit            *int
	Offset           *int
	MaxUsage         *int
	OlderThanDays    *int
}

type EnvStats struct {
	Total     int
	Available int
	Consumed  int
	Frozen    int
	Deleted   int
}

func OpenStore(path string) (*Store, error) {
	if path == "" {
		path = "miserver.db"
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS env_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_code TEXT NOT NULL,
			device_id TEXT NOT NULL,
			type TEXT NOT NULL,
			serial_backup_name TEXT NOT NULL,
			android_id TEXT NOT NULL,
			key TEXT NOT NULL,
			usage_count INTEGER NOT NULL DEFAULT 0,
			max_usage INTEGER NOT NULL DEFAULT 1,
			frozen INTEGER NOT NULL DEFAULT 0,
			consumed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddEnv(record EnvRecord, now time.Time) (int64, error) {
	if record.MaxUsage <= 0 {
		record.MaxUsage = 1
	}
	if now.IsZero() {
		now = time.Now()
	}
	res, err := s.db.Exec(
		`INSERT INTO env_records
		 (device_code, device_id, type, serial_backup_name, android_id, "key", usage_count, max_usage, frozen, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, 0, ?, ?)`,
		record.DeviceCode,
		record.DeviceID,
		record.Type,
		record.SerialBackupName,
		record.AndroidID,
		record.Key,
		record.MaxUsage,
		now.UTC(),
		now.UTC(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ConsumeEnv(filter EnvFilter, now time.Time) (*EnvRecord, error) {
	if now.IsZero() {
		now = time.Now()
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	where, args := envWhere(filter, false, now)
	query := `SELECT id, device_code, device_id, type, serial_backup_name, android_id, "key",
		usage_count, max_usage, frozen, consumed_at, created_at, updated_at, deleted_at
		FROM env_records ` + where + ` ORDER BY id ASC LIMIT 1`
	row := tx.QueryRow(query, args...)
	record, err := scanEnv(row)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		tx = nil
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(
		`UPDATE env_records
		 SET usage_count = 1, consumed_at = ?, updated_at = ?
		 WHERE id = ? AND consumed_at IS NULL AND deleted_at IS NULL AND frozen = 0`,
		now.UTC(),
		now.UTC(),
		record.ID,
	)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		tx = nil
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	record.UsageCount = 1
	consumedAt := now.UTC()
	record.ConsumedAt = &consumedAt
	return record, nil
}

func (s *Store) ListEnvs(filter EnvFilter) ([]EnvRecord, error) {
	where, args := envWhere(filter, true, time.Time{})
	query := `SELECT id, device_code, device_id, type, serial_backup_name, android_id, "key",
		usage_count, max_usage, frozen, consumed_at, created_at, updated_at, deleted_at
		FROM env_records ` + where + ` ORDER BY id ASC`
	if filter.Limit != nil && *filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, *filter.Limit)
	}
	if filter.Offset != nil && *filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, *filter.Offset)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []EnvRecord
	for rows.Next() {
		record, err := scanEnv(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Store) GetEnvByID(id int64) (*EnvRecord, error) {
	row := s.db.QueryRow(`SELECT id, device_code, device_id, type, serial_backup_name, android_id, "key",
		usage_count, max_usage, frozen, consumed_at, created_at, updated_at, deleted_at
		FROM env_records WHERE id = ?`, id)
	record, err := scanEnv(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return record, err
}

func (s *Store) SetEnvFrozen(id int64, frozen bool, now time.Time) error {
	value := 0
	if frozen {
		value = 1
	}
	_, err := s.db.Exec(`UPDATE env_records SET frozen = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, value, now.UTC(), id)
	return err
}

func (s *Store) DeleteEnv(id int64, now time.Time) error {
	_, err := s.db.Exec(`UPDATE env_records SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now.UTC(), now.UTC(), id)
	return err
}

func (s *Store) CleanEnv() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM env_records WHERE deleted_at IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) EnvStats() (EnvStats, error) {
	var stats EnvStats
	rows := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM env_records`, &stats.Total},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND frozen = 0 AND consumed_at IS NULL`, &stats.Available},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND consumed_at IS NOT NULL`, &stats.Consumed},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND frozen = 1`, &stats.Frozen},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NOT NULL`, &stats.Deleted},
	}
	for _, row := range rows {
		if err := s.db.QueryRow(row.query).Scan(row.dest); err != nil {
			return EnvStats{}, err
		}
	}
	return stats, nil
}

func envWhere(filter EnvFilter, includeStateFilters bool, now time.Time) (string, []any) {
	clauses := []string{"1 = 1"}
	args := []any{}
	add := func(column string, value string) {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	add("type", filter.Type)
	add("device_code", filter.DeviceCode)
	add("device_id", filter.DeviceID)
	add("serial_backup_name", filter.SerialBackupName)
	add("android_id", filter.AndroidID)
	add(`"key"`, filter.Key)
	if filter.MaxUsage != nil {
		clauses = append(clauses, "max_usage <= ?")
		args = append(args, *filter.MaxUsage)
	}
	if filter.OlderThanDays != nil && !now.IsZero() {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, now.Add(-time.Duration(*filter.OlderThanDays)*24*time.Hour).UTC())
	}
	if includeStateFilters {
		if filter.Frozen != nil {
			clauses = append(clauses, "frozen = ?")
			args = append(args, *filter.Frozen)
		}
	} else {
		clauses = append(clauses, "deleted_at IS NULL", "frozen = 0", "consumed_at IS NULL")
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

type envScanner interface {
	Scan(dest ...any) error
}

func scanEnv(scanner envScanner) (*EnvRecord, error) {
	var record EnvRecord
	var frozen int
	var consumedAt, deletedAt sql.NullTime
	if err := scanner.Scan(
		&record.ID,
		&record.DeviceCode,
		&record.DeviceID,
		&record.Type,
		&record.SerialBackupName,
		&record.AndroidID,
		&record.Key,
		&record.UsageCount,
		&record.MaxUsage,
		&frozen,
		&consumedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	record.Frozen = frozen != 0
	if consumedAt.Valid {
		t := consumedAt.Time
		record.ConsumedAt = &t
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		record.DeletedAt = &t
	}
	return &record, nil
}

func intFromAny(value any) (*int, bool) {
	switch v := value.(type) {
	case nil:
		return nil, false
	case float64:
		out := int(v)
		return &out, true
	case int:
		out := v
		return &out, true
	default:
		return nil, false
	}
}

func int64FromAny(value any) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid integer value %T", value)
	}
}
