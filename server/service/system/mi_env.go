package system

import (
	"errors"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MIEnvFilter struct {
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

type MIEnvStats struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
	Consumed  int64 `json:"consumed"`
	Frozen    int64 `json:"frozen"`
	Deleted   int64 `json:"deleted"`
	Unused    int64 `json:"unused"`
}

type MIEnvService struct{}

var MIEnvServiceApp = new(MIEnvService)

var ErrMIEnvUnavailable = errors.New("环境不存在或状态不可用")

// A make reservation prevents two device threads from restoring and modifying
// the same environment concurrently. It expires automatically because the
// compiled client has no explicit cancellation callback when device work fails.
const miEnvMakeReservationTTL = 2 * time.Hour

func (s *MIEnvService) Add(record model.SysMIEnvRecord, now time.Time) (uint, error) {
	if record.MaxUsage <= 0 {
		record.MaxUsage = 1
	}
	if record.MadeCount <= 0 {
		record.MadeCount = 1
	}
	if now.IsZero() {
		now = time.Now()
	}
	record.CreatedAt = now
	record.UpdatedAt = now
	var existing model.SysMIEnvRecord
	err := global.GVA_DB.Where(
		"device_code = ? AND device_id = ? AND type = ? AND serial_backup_name = ? AND android_id = ? AND env_key = ? AND deleted_at IS NULL",
		record.DeviceCode,
		record.DeviceID,
		record.Type,
		record.SerialBackupName,
		record.AndroidID,
		record.Key,
	).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if err := global.GVA_DB.Create(&record).Error; err != nil {
		return 0, err
	}
	return record.ID, nil
}

func (s *MIEnvService) Consume(filter MIEnvFilter, now time.Time) (*model.SysMIEnvRecord, error) {
	if now.IsZero() {
		now = time.Now()
	}
	var record model.SysMIEnvRecord
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		query := applyMIEnvFilter(tx, filter, false, now).Order(miEnvOrder(filter.Sort)).Clauses(clause.Locking{Strength: "UPDATE"})
		if err := query.First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		updates := map[string]any{
			"usage_count":         record.UsageCount + 1,
			"last_used_at":        now,
			"make_reserved_until": nil,
			"updated_at":          now,
		}
		if record.UsageCount+1 >= record.MaxUsage {
			updates["consumed_at"] = now
		}
		result := tx.Model(&model.SysMIEnvRecord{}).Where("id = ? AND usage_count = ? AND deleted_at IS NULL AND frozen = ?", record.ID, record.UsageCount, false).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			record = model.SysMIEnvRecord{}
			return nil
		}
		record.UsageCount++
		record.LastUsedAt = timePtr(now)
		record.MakeReservedUntil = nil
		if record.UsageCount >= record.MaxUsage {
			record.ConsumedAt = timePtr(now)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if record.ID == 0 {
		return nil, nil
	}
	return &record, nil
}

func (s *MIEnvService) List(filter MIEnvFilter) ([]model.SysMIEnvRecord, error) {
	var records []model.SysMIEnvRecord
	query := applyMIEnvFilter(global.GVA_DB, filter, true, time.Time{}).Order(miEnvOrder(filter.Sort))
	if filter.Limit != nil && *filter.Limit > 0 {
		query = query.Limit(*filter.Limit)
	}
	if filter.Offset != nil && *filter.Offset > 0 {
		query = query.Offset(*filter.Offset)
	}
	return records, query.Find(&records).Error
}

func (s *MIEnvService) Get(id uint) (*model.SysMIEnvRecord, error) {
	var record model.SysMIEnvRecord
	if err := global.GVA_DB.Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (s *MIEnvService) SetFrozen(id uint, frozen bool, now time.Time) error {
	updates := map[string]any{"frozen": frozen, "updated_at": now}
	if frozen {
		updates["make_reserved_until"] = nil
	}
	result := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	return activeMIEnvMutationError(result, id)
}

func (s *MIEnvService) SetFrozenByFilter(filter MIEnvFilter, frozen bool, now time.Time) (int64, error) {
	updates := map[string]any{"frozen": frozen, "updated_at": now}
	if frozen {
		updates["make_reserved_until"] = nil
	}
	result := applyMIEnvFilter(global.GVA_DB.Model(&model.SysMIEnvRecord{}), filter, true, now).Where("deleted_at IS NULL").Updates(updates)
	return result.RowsAffected, result.Error
}

func (s *MIEnvService) MarkMakeSuccess(id uint, now time.Time) (*model.SysMIEnvRecord, error) {
	if now.IsZero() {
		now = time.Now()
	}
	var record model.SysMIEnvRecord
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.SysMIEnvRecord{}).
			Where("id = ? AND deleted_at IS NULL AND frozen = ? AND usage_count = 0 AND make_reserved_until IS NOT NULL AND make_reserved_until > ?", id, false, now).
			Updates(map[string]any{
				"made_count":          gorm.Expr("made_count + ?", 1),
				"last_used_at":        now,
				"make_reserved_until": nil,
				"updated_at":          now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrMIEnvUnavailable
		}
		return tx.Where("id = ?", id).First(&record).Error
	})
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *MIEnvService) AdjustMadeCount(id uint, delta int, reset bool, now time.Time) error {
	updates := map[string]any{"updated_at": now}
	if reset {
		updates["made_count"] = 0
	} else {
		updates["made_count"] = gorm.Expr("CASE WHEN made_count + ? < 0 THEN 0 ELSE made_count + ? END", delta, delta)
	}
	result := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	return activeMIEnvMutationError(result, id)
}

func (s *MIEnvService) ListForMake(filter MIEnvFilter, now time.Time) (*model.SysMIEnvRecord, error) {
	if now.IsZero() {
		now = time.Now()
	}
	target := intValue(filter.MaxMadeCount, 3)
	cooldown := intValue(filter.CooldownDays, 1)
	if cooldown < 0 {
		cooldown = 0
	}
	filter.MinMadeCount = nil
	filter.MaxMadeCount = nil
	reservedUntil := now.Add(miEnvMakeReservationTTL)
	for attempt := 0; attempt < 3; attempt++ {
		var record model.SysMIEnvRecord
		reserved := false
		err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
			query := applyMIEnvFilter(tx, filter, true, now).
				Where("deleted_at IS NULL AND frozen = ? AND usage_count = 0", false).
				Where("made_count < ? AND (last_used_at IS NULL OR last_used_at <= ?)", target, now.Add(-time.Duration(cooldown)*24*time.Hour)).
				Where("make_reserved_until IS NULL OR make_reserved_until <= ?", now).
				Order("created_at ASC, made_count DESC, usage_count ASC, id ASC").
				Clauses(clause.Locking{Strength: "UPDATE"})
			if err := query.First(&record).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					record = model.SysMIEnvRecord{}
					return nil
				}
				return err
			}
			result := tx.Model(&model.SysMIEnvRecord{}).
				Where("id = ? AND deleted_at IS NULL AND frozen = ? AND usage_count = 0 AND (make_reserved_until IS NULL OR make_reserved_until <= ?)", record.ID, false, now).
				Update("make_reserved_until", reservedUntil)
			if result.Error != nil {
				return result.Error
			}
			reserved = result.RowsAffected == 1
			return nil
		})
		if err != nil {
			return nil, err
		}
		if record.ID == 0 {
			return nil, nil
		}
		if reserved {
			record.MakeReservedUntil = timePtr(reservedUntil)
			return &record, nil
		}
	}
	return nil, nil
}

func (s *MIEnvService) Delete(id uint, now time.Time) error {
	result := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]any{"deleted_at": now, "make_reserved_until": nil, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMIEnvUnavailable
	}
	return nil
}

func (s *MIEnvService) DeleteByFilter(filter MIEnvFilter, now time.Time) (int64, error) {
	result := applyMIEnvFilter(global.GVA_DB.Model(&model.SysMIEnvRecord{}), filter, true, now).Where("deleted_at IS NULL").Updates(map[string]any{"deleted_at": now, "make_reserved_until": nil, "updated_at": now})
	return result.RowsAffected, result.Error
}

func (s *MIEnvService) Clean(olderThan *int, now time.Time) (int64, error) {
	if olderThan == nil {
		result := global.GVA_DB.Where("deleted_at IS NOT NULL").Delete(&model.SysMIEnvRecord{})
		return result.RowsAffected, result.Error
	}
	days := *olderThan
	if days < 0 {
		days = 0
	}
	result := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where("deleted_at IS NULL AND created_at <= ?", now.Add(-time.Duration(days)*24*time.Hour)).Updates(map[string]any{"deleted_at": now, "make_reserved_until": nil, "updated_at": now})
	return result.RowsAffected, result.Error
}

func (s *MIEnvService) Stats() (MIEnvStats, error) {
	var out MIEnvStats
	now := time.Now()
	queries := []struct {
		where string
		args  []any
		dest  *int64
	}{
		{"1 = 1", nil, &out.Total},
		{"deleted_at IS NULL AND frozen = ? AND usage_count < max_usage AND (make_reserved_until IS NULL OR make_reserved_until <= ?)", []any{false, now}, &out.Available},
		{"deleted_at IS NULL AND usage_count > 0", nil, &out.Consumed},
		{"deleted_at IS NULL AND frozen = ?", []any{true}, &out.Frozen},
		{"deleted_at IS NOT NULL", nil, &out.Deleted},
		{"deleted_at IS NULL AND usage_count = 0", nil, &out.Unused},
	}
	for _, item := range queries {
		if err := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where(item.where, item.args...).Count(item.dest).Error; err != nil {
			return MIEnvStats{}, err
		}
	}
	return out, nil
}

func (s *MIEnvService) StatsByType() (map[string]int64, error) {
	var rows []struct {
		Type  string
		Count int64
	}
	err := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Select("type, COUNT(*) AS count").Where("deleted_at IS NULL").Group("type").Order("type").Scan(&rows).Error
	out := map[string]int64{}
	for _, row := range rows {
		out[row.Type] = row.Count
	}
	return out, err
}

func (s *MIEnvService) StatsMakeProgress() (map[string]int64, error) {
	var rows []struct {
		MadeCount int
		Count     int64
	}
	err := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Select("made_count, COUNT(*) AS count").Where("deleted_at IS NULL").Group("made_count").Order("made_count").Scan(&rows).Error
	out := map[string]int64{}
	for _, row := range rows {
		out[strconv.Itoa(row.MadeCount)] = row.Count
	}
	return out, err
}

func applyMIEnvFilter(db *gorm.DB, filter MIEnvFilter, includeState bool, now time.Time) *gorm.DB {
	add := func(column, value string) {
		if value != "" {
			db = db.Where(column+" = ?", value)
		}
	}
	add("type", filter.Type)
	add("device_code", filter.DeviceCode)
	add("device_id", filter.DeviceID)
	add("serial_backup_name", filter.SerialBackupName)
	add("android_id", filter.AndroidID)
	add("env_key", filter.Key)
	if filter.MaxUsage != nil {
		db = db.Where("usage_count <= ?", *filter.MaxUsage)
	}
	if filter.MinUsage != nil {
		db = db.Where("usage_count >= ?", *filter.MinUsage)
	}
	if filter.MinMadeCount != nil {
		db = db.Where("made_count >= ?", *filter.MinMadeCount)
	}
	if filter.MaxMadeCount != nil {
		db = db.Where("made_count <= ?", *filter.MaxMadeCount)
	}
	if !now.IsZero() {
		if filter.OlderThanDays != nil {
			db = db.Where("created_at <= ?", now.Add(-time.Duration(*filter.OlderThanDays)*24*time.Hour))
		}
		if filter.MinDays != nil {
			db = db.Where("created_at >= ?", now.Add(-time.Duration(*filter.MinDays)*24*time.Hour))
		}
		if filter.MaxDays != nil {
			db = db.Where("created_at <= ?", now.Add(-time.Duration(*filter.MaxDays)*24*time.Hour))
		}
	}
	if includeState {
		if filter.Frozen != nil {
			db = db.Where("frozen = ?", *filter.Frozen != 0)
		}
	} else {
		db = db.Where("deleted_at IS NULL AND frozen = ? AND usage_count < max_usage", false)
		if !now.IsZero() {
			db = db.Where("make_reserved_until IS NULL OR make_reserved_until <= ?", now)
		}
	}
	return db
}

func activeMIEnvMutationError(result *gorm.DB, id uint) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	var count int64
	if err := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where("id = ? AND deleted_at IS NULL", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrMIEnvUnavailable
	}
	return nil
}

func miEnvOrder(sortMode string) string {
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

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
func timePtr(value time.Time) *time.Time { return &value }
