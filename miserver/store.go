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
	MadeCount        int
	Frozen           bool
	ConsumedAt       *time.Time
	LastUsedAt       *time.Time
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
	MinUsage         *int
	MinMadeCount     *int
	MaxMadeCount     *int
	OlderThanDays    *int
	MinDays          *int
	MaxDays          *int
	CooldownDays     *int
	Sort             string
}

type EnvStats struct {
	Total     int
	Available int
	Consumed  int
	Frozen    int
	Deleted   int
	Unused    int
}

type LicenseRecord struct {
	DeviceID  string
	StartedAt time.Time
	ExpiresAt time.Time
}

type UploadRecord struct {
	Device, CurrentTime, Phone, Account, Password string
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
			made_count INTEGER NOT NULL DEFAULT 1,
			frozen INTEGER NOT NULL DEFAULT 0,
			consumed_at DATETIME,
			last_used_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS licenses (
			device_id TEXT PRIMARY KEY,
			started_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS account_uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device TEXT NOT NULL,
			current_time TEXT NOT NULL,
			phone TEXT NOT NULL,
			account TEXT NOT NULL,
			password TEXT NOT NULL,
			uploaded_at DATETIME NOT NULL,
			UNIQUE(device, account, password)
		)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	// Keep existing local databases compatible with the latest environment
	// lifecycle fields. SQLite returns an error when the column already exists;
	// that is intentionally ignored for these additive migrations.
	for _, stmt := range []string{
		`ALTER TABLE env_records ADD COLUMN made_count INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE env_records ADD COLUMN last_used_at DATETIME`,
	} {
		_, _ = s.db.Exec(stmt)
	}
	return nil
}

func (s *Store) SaveUpload(record UploadRecord, now time.Time) (bool, error) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO account_uploads(device, current_time, phone, account, password, uploaded_at) VALUES (?, ?, ?, ?, ?, ?)`, record.Device, record.CurrentTime, record.Phone, record.Account, record.Password, now.UTC())
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Store) GetOrCreateLicense(deviceID string, now time.Time, days int) (LicenseRecord, error) {
	var record LicenseRecord
	err := s.db.QueryRow(`SELECT device_id, started_at, expires_at FROM licenses WHERE device_id = ?`, deviceID).Scan(&record.DeviceID, &record.StartedAt, &record.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		if days <= 0 {
			days = 30
		}
		record = LicenseRecord{DeviceID: deviceID, StartedAt: now.UTC(), ExpiresAt: now.UTC().Add(time.Duration(days) * 24 * time.Hour)}
		_, err = s.db.Exec(`INSERT INTO licenses(device_id, started_at, expires_at, updated_at) VALUES (?, ?, ?, ?)`, record.DeviceID, record.StartedAt, record.ExpiresAt, now.UTC())
	}
	return record, err
}

func (s *Store) AddEnv(record EnvRecord, now time.Time) (int64, error) {
	if record.MaxUsage <= 0 {
		record.MaxUsage = 1
	}
	if now.IsZero() {
		now = time.Now()
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var existingID int64
	err = tx.QueryRow(`SELECT id FROM env_records WHERE type = ? AND device_id = ? AND serial_backup_name = ? AND deleted_at IS NULL ORDER BY id LIMIT 1`, record.Type, record.DeviceID, record.SerialBackupName).Scan(&existingID)
	if err == nil {
		_, err = tx.Exec(`UPDATE env_records SET device_code = ?, device_id = ?, android_id = ?, "key" = ?, updated_at = ? WHERE id = ?`, record.DeviceCode, record.DeviceID, record.AndroidID, record.Key, now.UTC(), existingID)
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return existingID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := tx.Exec(
		`INSERT INTO env_records
			(device_code, device_id, type, serial_backup_name, android_id, "key", usage_count, max_usage, made_count, frozen, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 0, ?, 1, 0, ?, ?)`,
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
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
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
		usage_count, max_usage, made_count, frozen, consumed_at, last_used_at, created_at, updated_at, deleted_at
		FROM env_records ` + where + ` ORDER BY ` + envOrder(filter.Sort) + ` LIMIT 1`
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
		SET usage_count = usage_count + 1,
			consumed_at = CASE WHEN usage_count + 1 >= max_usage THEN ? ELSE consumed_at END,
			last_used_at = ?, updated_at = ?
		WHERE id = ? AND usage_count = ? AND deleted_at IS NULL AND frozen = 0`,
		now.UTC(),
		now.UTC(),
		now.UTC(),
		record.ID,
		record.UsageCount,
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
	record.UsageCount++
	lastUsedAt := now.UTC()
	record.LastUsedAt = &lastUsedAt
	if record.UsageCount >= record.MaxUsage {
		record.ConsumedAt = &lastUsedAt
	}
	return record, nil
}

func (s *Store) ListEnvs(filter EnvFilter) ([]EnvRecord, error) {
	where, args := envWhere(filter, true, time.Time{})
	query := `SELECT id, device_code, device_id, type, serial_backup_name, android_id, "key",
		usage_count, max_usage, made_count, frozen, consumed_at, last_used_at, created_at, updated_at, deleted_at
		FROM env_records ` + where + ` ORDER BY ` + envOrder(filter.Sort)
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

func envOrder(sortMode string) string {
	switch sortMode {
	case "创建时间优先":
		return "created_at ASC, made_count DESC, usage_count ASC, id ASC"
	case "制作次数优先":
		return "made_count DESC, created_at ASC, usage_count ASC, id ASC"
	case "使用次数优先":
		return "usage_count ASC, created_at ASC, made_count DESC, id ASC"
	case "id DESC":
		return "id DESC"
	default:
		return "id ASC"
	}
}

func (s *Store) GetEnvByID(id int64) (*EnvRecord, error) {
	row := s.db.QueryRow(`SELECT id, device_code, device_id, type, serial_backup_name, android_id, "key",
		usage_count, max_usage, made_count, frozen, consumed_at, last_used_at, created_at, updated_at, deleted_at
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

func (s *Store) SetEnvFrozenByFilter(filter EnvFilter, frozen bool, now time.Time) (int64, error) {
	where, args := envWhere(filter, true, now)
	where += ` AND deleted_at IS NULL`
	value := 0
	if frozen {
		value = 1
	}
	queryArgs := []any{value, now.UTC()}
	queryArgs = append(queryArgs, args...)
	res, err := s.db.Exec(`UPDATE env_records SET frozen = ?, updated_at = ? `+where, queryArgs...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) MarkMakeSuccess(id int64, now time.Time) error {
	_, err := s.db.Exec(`UPDATE env_records SET made_count = made_count + 1, last_used_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now.UTC(), now.UTC(), id)
	return err
}

func (s *Store) AdjustMadeCount(id int64, delta int, reset bool, now time.Time) error {
	if reset {
		_, err := s.db.Exec(`UPDATE env_records SET made_count = 0, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now.UTC(), id)
		return err
	}
	_, err := s.db.Exec(`UPDATE env_records SET made_count = MAX(0, made_count + ?), updated_at = ? WHERE id = ? AND deleted_at IS NULL`, delta, now.UTC(), id)
	return err
}

func (s *Store) ListForMake(filter EnvFilter, now time.Time) (*EnvRecord, error) {
	// /get_env_for_make selects without consuming usage count. It only excludes
	// frozen/deleted rows and applies the requested make target/cooldown.
	target := defaultInt(filter.MaxMadeCount, 3)
	filter.MinMadeCount = nil
	filter.MaxMadeCount = nil
	where, args := envWhere(filter, true, now)
	where += ` AND deleted_at IS NULL AND frozen = 0`
	cooldown := defaultInt(filter.CooldownDays, 1)
	where += ` AND made_count < ? AND (last_used_at IS NULL OR last_used_at <= ?)`
	args = append(args, target, now.Add(-time.Duration(cooldown)*24*time.Hour).UTC())
	query := `SELECT id, device_code, device_id, type, serial_backup_name, android_id, "key",
		usage_count, max_usage, made_count, frozen, consumed_at, last_used_at, created_at, updated_at, deleted_at
		FROM env_records ` + where + ` ORDER BY created_at ASC, made_count DESC, usage_count ASC, id ASC LIMIT 1`
	record, err := scanEnv(s.db.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return record, err
}

func (s *Store) DeleteEnv(id int64, now time.Time) error {
	_, err := s.db.Exec(`UPDATE env_records SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, now.UTC(), now.UTC(), id)
	return err
}

func (s *Store) DeleteEnvsByFilter(filter EnvFilter, now time.Time) (int64, error) {
	where, args := envWhere(filter, true, now)
	where += ` AND deleted_at IS NULL`
	queryArgs := []any{now.UTC(), now.UTC()}
	queryArgs = append(queryArgs, args...)
	res, err := s.db.Exec(`UPDATE env_records SET deleted_at = ?, updated_at = ? `+where, queryArgs...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) CleanEnv() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM env_records WHERE deleted_at IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) CleanEnvOlderThan(days int, now time.Time) (int64, error) {
	if days < 0 {
		days = 0
	}
	res, err := s.db.Exec(`UPDATE env_records SET deleted_at = ?, updated_at = ? WHERE deleted_at IS NULL AND created_at <= ?`, now.UTC(), now.UTC(), now.Add(-time.Duration(days)*24*time.Hour).UTC())
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
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND frozen = 0`, &stats.Available},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND usage_count > 0`, &stats.Consumed},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND frozen = 1`, &stats.Frozen},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NOT NULL`, &stats.Deleted},
		{`SELECT COUNT(*) FROM env_records WHERE deleted_at IS NULL AND usage_count = 0`, &stats.Unused},
	}
	for _, row := range rows {
		if err := s.db.QueryRow(row.query).Scan(row.dest); err != nil {
			return EnvStats{}, err
		}
	}
	return stats, nil
}

func (s *Store) StatsByType() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT type, COUNT(*) FROM env_records WHERE deleted_at IS NULL GROUP BY type ORDER BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, err
		}
		out[typ] = count
	}
	return out, rows.Err()
}

func (s *Store) StatsMakeProgress() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT CAST(made_count AS TEXT), COUNT(*) FROM env_records WHERE deleted_at IS NULL GROUP BY made_count ORDER BY made_count`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var progress string
		var count int
		if err := rows.Scan(&progress, &count); err != nil {
			return nil, err
		}
		out[progress] = count
	}
	return out, rows.Err()
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
		clauses = append(clauses, "usage_count <= ?")
		args = append(args, *filter.MaxUsage)
	}
	if filter.MinUsage != nil {
		clauses = append(clauses, "usage_count >= ?")
		args = append(args, *filter.MinUsage)
	}
	if filter.MinMadeCount != nil {
		clauses = append(clauses, "made_count >= ?")
		args = append(args, *filter.MinMadeCount)
	}
	if filter.MaxMadeCount != nil {
		clauses = append(clauses, "made_count <= ?")
		args = append(args, *filter.MaxMadeCount)
	}
	if filter.OlderThanDays != nil && !now.IsZero() {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, now.Add(-time.Duration(*filter.OlderThanDays)*24*time.Hour).UTC())
	}
	if filter.MinDays != nil && !now.IsZero() {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, now.Add(-time.Duration(*filter.MinDays)*24*time.Hour).UTC())
	}
	if filter.MaxDays != nil && !now.IsZero() {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, now.Add(-time.Duration(*filter.MaxDays)*24*time.Hour).UTC())
	}
	if includeStateFilters {
		if filter.Frozen != nil {
			clauses = append(clauses, "frozen = ?")
			args = append(args, *filter.Frozen)
		}
	} else {
		clauses = append(clauses, "deleted_at IS NULL", "frozen = 0")
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

type envScanner interface {
	Scan(dest ...any) error
}

func scanEnv(scanner envScanner) (*EnvRecord, error) {
	var record EnvRecord
	var frozen int
	var consumedAt, lastUsedAt, deletedAt sql.NullTime
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
		&record.MadeCount,
		&frozen,
		&consumedAt,
		&lastUsedAt,
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
	if lastUsedAt.Valid {
		t := lastUsedAt.Time
		record.LastUsedAt = &t
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

func defaultInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
