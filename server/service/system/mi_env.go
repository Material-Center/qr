package system

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
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

// Import inserts a record while preserving the source system's historical
// timestamps and counters. It is intentionally separate from Add: client
// uploads should continue to use the current-time semantics of Add, whereas
// migration tools must be able to replay old records safely.
func (s *MIEnvService) Import(record model.SysMIEnvRecord, now time.Time) (uint, bool, error) {
	if record.DeviceCode == "" || record.DeviceID == "" || record.Type == "" || record.SerialBackupName == "" || record.AndroidID == "" || record.Key == "" {
		return 0, false, errors.New("environment identity fields are required")
	}
	if record.CreatedAt.IsZero() {
		return 0, false, errors.New("created_at is required")
	}
	if record.MaxUsage <= 0 {
		record.MaxUsage = 1
	}
	if record.MadeCount < 0 {
		record.MadeCount = 0
	}
	if record.UsageCount < 0 {
		record.UsageCount = 0
	}
	if now.IsZero() {
		now = time.Now()
	}
	record.UpdatedAt = now
	record.DeletedAt = gorm.DeletedAt{}
	record.MakeReservedUntil = nil

	var existing model.SysMIEnvRecord
	err := global.GVA_DB.Where(
		"device_code = ? AND device_id = ? AND type = ? AND serial_backup_name = ? AND android_id = ? AND env_key = ? AND deleted_at IS NULL",
		record.DeviceCode, record.DeviceID, record.Type, record.SerialBackupName, record.AndroidID, record.Key,
	).First(&existing).Error
	if err == nil {
		// Keep imports idempotent. Refresh only fields owned by the source
		// record; do not change the target primary key.
		updates := map[string]any{
			"usage_count": record.UsageCount, "max_usage": record.MaxUsage,
			"made_count": record.MadeCount, "frozen": record.Frozen,
			"consumed_at": record.ConsumedAt, "last_used_at": record.LastUsedAt,
			"created_at": record.CreatedAt, "updated_at": now,
		}
		if updateErr := global.GVA_DB.Model(&model.SysMIEnvRecord{}).Where("id = ?", existing.ID).Updates(updates).Error; updateErr != nil {
			return 0, false, updateErr
		}
		return existing.ID, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}
	if err := global.GVA_DB.Create(&record).Error; err != nil {
		return 0, false, err
	}
	return record.ID, true, nil
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

func (s *MIEnvService) List(filter MIEnvFilter, now time.Time) ([]model.SysMIEnvRecord, error) {
	if now.IsZero() {
		now = time.Now()
	}
	var records []model.SysMIEnvRecord
	query := applyMIEnvFilter(global.GVA_DB, filter, true, now).Order(miEnvOrder(filter.Sort))
	if filter.Limit != nil && *filter.Limit > 0 {
		query = query.Limit(*filter.Limit)
	}
	if filter.Offset != nil && *filter.Offset > 0 {
		query = query.Offset(*filter.Offset)
	}
	return records, query.Find(&records).Error
}

// ListForAdmin returns active environment records for the management page.
// It deliberately uses a separate filter contract from the compiled-client
// protocol so admin pagination and date-range semantics cannot affect clients.
func (s *MIEnvService) ListForAdmin(req systemReq.MIEnvAdminList) ([]model.SysMIEnvRecord, int64, error) {
	query, err := applyMIEnvAdminFilter(global.GVA_DB.Model(&model.SysMIEnvRecord{}), req)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	var list []model.SysMIEnvRecord
	err = query.Order("updated_at DESC").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ListTypesForAdmin returns active environment types under the selected device
// and device-group scope for the admin type autocomplete.
func (s *MIEnvService) ListTypesForAdmin(req systemReq.MIEnvAdminList) ([]string, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	req.DeviceID = ""
	req.Type = ""
	req.MinUsage = nil
	req.MaxUsage = nil
	req.MinMadeCount = nil
	req.MaxMadeCount = nil
	req.Status = ""
	req.UpdatedAtStart = ""
	req.UpdatedAtEnd = ""
	query, err := applyMIEnvAdminFilter(global.GVA_DB.Model(&model.SysMIEnvRecord{}), req)
	if err != nil {
		return nil, err
	}
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	var types []string
	err = query.Where("type <> ''").Distinct("type").Order("type ASC").Limit(200).Pluck("type", &types).Error
	return types, err
}

// DeleteAllForAdmin soft-deletes every active record matching the current
// admin filters. Soft deletion preserves protocol statistics and auditability.
func (s *MIEnvService) DeleteAllForAdmin(req systemReq.MIEnvAdminList, now time.Time) (int64, error) {
	if now.IsZero() {
		now = time.Now()
	}
	query, err := applyMIEnvAdminFilter(global.GVA_DB.Model(&model.SysMIEnvRecord{}), req)
	if err != nil {
		return 0, err
	}
	result := query.Updates(map[string]any{
		"deleted_at":          now,
		"make_reserved_until": nil,
		"updated_at":          now,
	})
	return result.RowsAffected, result.Error
}

// DeleteSelectedForAdmin soft-deletes only the explicitly selected active records.
func (s *MIEnvService) DeleteSelectedForAdmin(ids []uint, now time.Time) (int64, error) {
	if global.GVA_DB == nil {
		return 0, errors.New("数据库未初始化")
	}
	if len(ids) == 0 {
		return 0, errors.New("请选择要删除的环境")
	}
	if len(ids) > 200 {
		return 0, errors.New("单次最多删除 200 条环境")
	}
	uniqueIDs := make([]uint, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			return 0, errors.New("环境 ID 无效")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if now.IsZero() {
		now = time.Now()
	}
	result := global.GVA_DB.Model(&model.SysMIEnvRecord{}).
		Where("id IN ? AND deleted_at IS NULL", uniqueIDs).
		Updates(map[string]any{
			"deleted_at":          now,
			"make_reserved_until": nil,
			"updated_at":          now,
		})
	return result.RowsAffected, result.Error
}

func applyMIEnvAdminFilter(db *gorm.DB, req systemReq.MIEnvAdminList) (*gorm.DB, error) {
	if db == nil {
		return nil, errors.New("数据库未初始化")
	}
	db = db.Where("deleted_at IS NULL")
	if deviceID := strings.TrimSpace(req.DeviceID); deviceID != "" {
		db = db.Where("device_id = ?", deviceID)
	}
	if req.GroupID != nil && *req.GroupID != 0 {
		deviceIDs := global.GVA_DB.Model(&model.SysDeviceConfig{}).
			Select("device_id").
			Where("group_id = ?", *req.GroupID)
		db = db.Where("device_id IN (?)", deviceIDs)
	} else if req.Ungrouped {
		db = db.Where(`NOT EXISTS (
			SELECT 1 FROM sys_device_configs AS device_config
			WHERE device_config.deleted_at IS NULL
				AND device_config.device_id = sys_mi_env_records.device_id
				AND device_config.group_id IS NOT NULL
				AND device_config.group_id <> 0
		)`)
	}
	if envType := strings.TrimSpace(req.Type); envType != "" {
		db = db.Where("type = ?", envType)
	}
	if err := validateMIEnvAdminRange("使用次数", req.MinUsage, req.MaxUsage); err != nil {
		return nil, err
	}
	if err := validateMIEnvAdminRange("制作次数", req.MinMadeCount, req.MaxMadeCount); err != nil {
		return nil, err
	}
	if req.MinUsage != nil {
		db = db.Where("usage_count >= ?", *req.MinUsage)
	}
	if req.MaxUsage != nil {
		db = db.Where("usage_count <= ?", *req.MaxUsage)
	}
	if req.MinMadeCount != nil {
		db = db.Where("made_count >= ?", *req.MinMadeCount)
	}
	if req.MaxMadeCount != nil {
		db = db.Where("made_count <= ?", *req.MaxMadeCount)
	}
	switch status := strings.TrimSpace(req.Status); status {
	case "", "all":
	case "normal":
		db = db.Where("frozen = ?", false)
	case "frozen":
		db = db.Where("frozen = ?", true)
	default:
		return nil, errors.New("状态仅支持 normal 或 frozen")
	}
	start, err := parseMIEnvAdminTime(req.UpdatedAtStart, false)
	if err != nil {
		return nil, err
	}
	end, err := parseMIEnvAdminTime(req.UpdatedAtEnd, true)
	if err != nil {
		return nil, err
	}
	if start != nil && end != nil && start.After(*end) {
		return nil, errors.New("更新时间开始值不能晚于结束值")
	}
	if start != nil {
		db = db.Where("updated_at >= ?", *start)
	}
	if end != nil {
		db = db.Where("updated_at <= ?", *end)
	}
	return db, nil
}

func validateMIEnvAdminRange(name string, minValue, maxValue *int) error {
	if minValue != nil && *minValue < 0 || maxValue != nil && *maxValue < 0 {
		return errors.New(name + "不能小于 0")
	}
	if minValue != nil && maxValue != nil && *minValue > *maxValue {
		return errors.New(name + "最小值不能大于最大值")
	}
	return nil
}

func parseMIEnvAdminTime(value string, endOfDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && endOfDay {
			parsed = parsed.Add(24*time.Hour - time.Nanosecond)
		}
		return &parsed, nil
	}
	return nil, errors.New("更新时间格式无效")
}

func (s *MIEnvService) Get(id uint) (*model.SysMIEnvRecord, error) {
	var record model.SysMIEnvRecord
	if err := global.GVA_DB.Unscoped().Where("id = ?", id).First(&record).Error; err != nil {
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
		result := global.GVA_DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&model.SysMIEnvRecord{})
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
		unscoped bool
		where    string
		args     []any
		dest     *int64
	}{
		{true, "1 = 1", nil, &out.Total},
		{false, "deleted_at IS NULL AND frozen = ? AND usage_count < max_usage AND (make_reserved_until IS NULL OR make_reserved_until <= ?)", []any{false, now}, &out.Available},
		{false, "deleted_at IS NULL AND usage_count > 0", nil, &out.Consumed},
		{false, "deleted_at IS NULL AND frozen = ?", []any{true}, &out.Frozen},
		{true, "deleted_at IS NOT NULL", nil, &out.Deleted},
		{false, "deleted_at IS NULL AND usage_count = 0", nil, &out.Unused},
	}
	for _, item := range queries {
		query := global.GVA_DB.Model(&model.SysMIEnvRecord{})
		if item.unscoped {
			query = query.Unscoped()
		}
		if err := query.Where(item.where, item.args...).Count(item.dest).Error; err != nil {
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
