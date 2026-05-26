package system

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const qqCacheExportIniMaxIDs = 100
const (
	qqCacheServiceRoleSuperAdmin = uint(888)
	qqCacheServiceRoleAdmin      = uint(100)
)

const (
	qqCacheInternalToolActionCreated = "created"
	qqCacheInternalToolActionSkipped = "skipped"
	qqCacheInternalToolActionUpdated = "updated"
)

type QQCacheService struct{}
type qqCacheBillingSettleResult struct {
	SettledAt    time.Time
	SettledCount int64
}

var QQCacheServiceApp = new(QQCacheService)

func (s *QQCacheService) UploadByApp(userID uint, req systemReq.QQCacheUpload) (system.SysQQCacheRecord, error) {
	_ = userID
	return s.uploadRecord(req)
}

func (s *QQCacheService) InternalToolFindQQCacheByQQNum(qqNum string) (system.SysQQCacheRecord, bool, error) {
	qqNum = strings.TrimSpace(qqNum)
	if qqNum == "" {
		return system.SysQQCacheRecord{}, false, errors.New("qq账号不能为空")
	}
	var record system.SysQQCacheRecord
	if err := global.GVA_DB.Unscoped().Where("qq_num = ?", qqNum).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return system.SysQQCacheRecord{}, false, nil
		}
		return system.SysQQCacheRecord{}, false, err
	}
	return record, true, nil
}

func (s *QQCacheService) InternalToolImportQQCache(req systemReq.InternalToolQQCacheImport) (system.SysQQCacheRecord, string, error) {
	qqNum := strings.TrimSpace(req.QQNum)
	iniText := strings.TrimSpace(req.INI)
	if qqNum == "" {
		return system.SysQQCacheRecord{}, "", errors.New("qq账号不能为空")
	}
	if iniText == "" {
		return system.SysQQCacheRecord{}, "", errors.New("缓存内容不能为空")
	}
	clientVersion := extractQQCacheINIValue(iniText, "clientVersion")

	var record system.SysQQCacheRecord
	action := qqCacheInternalToolActionSkipped
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Unscoped().Where("qq_num = ?", qqNum).First(&record).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if !req.Force {
				return nil
			}
			updates := map[string]any{
				"qq_pwd":         strings.TrimSpace(req.QQPwd),
				"ini":            iniText,
				"client_version": clientVersion,
				"deleted_at":     nil,
				"updated_at":     time.Now(),
			}
			if phone := strings.TrimSpace(req.Phone); phone != "" {
				updates["phone"] = phone
			}
			if deviceID := strings.TrimSpace(req.DeviceID); deviceID != "" {
				updates["device_id"] = deviceID
			}
			if err := tx.Unscoped().Model(&record).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("qq_num = ?", qqNum).First(&record).Error; err != nil {
				return err
			}
			action = qqCacheInternalToolActionUpdated
			return nil
		}

		record = system.SysQQCacheRecord{
			Phone:         trimToPtr(req.Phone),
			QQNum:         qqNum,
			QQPwd:         strings.TrimSpace(req.QQPwd),
			ClientVersion: clientVersion,
			INI:           stringPtr(iniText),
			DeviceID:      trimToPtr(req.DeviceID),
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		action = qqCacheInternalToolActionCreated
		return nil
	})
	if err != nil {
		return system.SysQQCacheRecord{}, "", err
	}
	return record, action, nil
}

func (s *QQCacheService) AdminImportQQCache(req systemReq.InternalToolQQCacheImport) (system.SysQQCacheRecord, string, error) {
	req.Force = true
	return s.InternalToolImportQQCache(req)
}

func (s *QQCacheService) UploadPhoneRegister(req systemReq.QQCacheUpload) (systemRes system.SysQQCacheRecord, task system.SysPhoneRegisterTask, err error) {
	if strings.TrimSpace(req.DeviceID) == "" {
		return system.SysQQCacheRecord{}, system.SysPhoneRegisterTask{}, errors.New("deviceId不能为空")
	}
	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		record, upErr := s.uploadRecordTx(tx, req)
		if upErr != nil {
			return upErr
		}
		systemRes = record
		phoneTaskService := &PhoneRegisterTaskService{}
		var completedTask system.SysPhoneRegisterTask
		var taskErr error
		if req.TaskID != 0 {
			completedTask, taskErr = phoneTaskService.AttachOpenAPICacheTx(tx, strings.TrimSpace(req.DeviceID), req.TaskID, record.ID, record.QQNum)
		} else {
			completedTask, taskErr = phoneTaskService.CompleteTaskAfterQQCacheUploadTx(tx, strings.TrimSpace(req.DeviceID), record.ID, record.QQNum)
		}
		if taskErr != nil {
			return taskErr
		}
		task = completedTask
		return nil
	})
	if err == nil {
		_ = (&DeviceService{}).MarkOffline(strings.TrimSpace(req.DeviceID))
	}
	return systemRes, task, err
}

func (s *QQCacheService) uploadRecord(req systemReq.QQCacheUpload) (system.SysQQCacheRecord, error) {
	qqNum := strings.TrimSpace(req.QQNum)
	iniText := strings.TrimSpace(req.INI)
	if qqNum == "" {
		return system.SysQQCacheRecord{}, errors.New("qq账号不能为空")
	}
	if iniText == "" {
		return system.SysQQCacheRecord{}, errors.New("缓存内容不能为空")
	}

	record := system.SysQQCacheRecord{}
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var upErr error
		record, upErr = s.uploadRecordTx(tx, req)
		return upErr
	})
	return record, err
}

func (s *QQCacheService) uploadRecordTx(tx *gorm.DB, req systemReq.QQCacheUpload) (system.SysQQCacheRecord, error) {
	qqNum := strings.TrimSpace(req.QQNum)
	iniText := strings.TrimSpace(req.INI)
	now := time.Now()
	phone := trimToPtr(req.Phone)
	deviceID := trimToPtr(req.DeviceID)
	clientVersion := extractQQCacheINIValue(iniText, "clientVersion")
	entity := system.SysQQCacheRecord{
		Phone:         phone,
		QQNum:         qqNum,
		QQPwd:         strings.TrimSpace(req.QQPwd),
		ClientVersion: clientVersion,
		INI:           stringPtr(iniText),
		DeviceID:      deviceID,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "qq_num"}},
		DoUpdates: clause.Assignments(map[string]any{
			"phone":             phone,
			"qq_pwd":            strings.TrimSpace(req.QQPwd),
			"client_version":    clientVersion,
			"ini":               iniText,
			"device_id":         deviceID,
			"extractor":         nil,
			"extract_record_id": nil,
			"extraction_at":     nil,
			"updated_at":        now,
			"deleted_at":        nil,
		}),
	}).Create(&entity).Error; err != nil {
		return system.SysQQCacheRecord{}, err
	}
	var record system.SysQQCacheRecord
	if err := tx.Where("qq_num = ?", qqNum).First(&record).Error; err != nil {
		return system.SysQQCacheRecord{}, err
	}
	return record, nil
}

func (s *QQCacheService) ExtractByApp(userID uint, req systemReq.QQCacheExtract) (system.SysQQCacheRecord, error) {
	qqNum := strings.TrimSpace(req.QQNum)
	if qqNum == "" {
		return system.SysQQCacheRecord{}, errors.New("qq账号不能为空")
	}
	record := system.SysQQCacheRecord{}
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("qq_num = ?", qqNum).First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("未找到该qq缓存")
			}
			return err
		}
		if record.INI == nil || strings.TrimSpace(*record.INI) == "" {
			return errors.New("该账号缓存为空")
		}
		if record.Extractor != nil && *record.Extractor != userID {
			return errors.New("该账号已被其他账号提取")
		}
		now := time.Now()
		updates := map[string]any{
			"extractor":         userID,
			"extract_record_id": record.ID,
			"extraction_at":     now,
		}
		rsp := tx.Model(&system.SysQQCacheRecord{}).
			Where("id = ? AND (extractor IS NULL OR extractor = ?)", record.ID, userID).
			Updates(updates)
		if rsp.Error != nil {
			return rsp.Error
		}
		if rsp.RowsAffected == 0 {
			var latest system.SysQQCacheRecord
			if err := tx.Where("id = ?", record.ID).First(&latest).Error; err != nil {
				return err
			}
			if latest.Extractor != nil && *latest.Extractor != userID {
				return errors.New("该账号已被其他账号提取")
			}
			return errors.New("提取失败，请稍后重试")
		}
		if err := tx.Where("id = ?", record.ID).First(&record).Error; err != nil {
			return err
		}
		return nil
	})
	return record, err
}

func (s *QQCacheService) ListForAdmin(req systemReq.QQCacheList) (list []system.SysQQCacheRecord, total int64, err error) {
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 10
	}
	db := applyQQCacheListFilters(global.GVA_DB.Model(&system.SysQQCacheRecord{}), qqCacheListFilter{
		QQNum:          req.QQNum,
		ClientVersion:  req.ClientVersion,
		DeviceID:       req.DeviceID,
		ExtractorID:    req.ExtractorID,
		Extracted:      req.Extracted,
		CreatedAtStart: req.CreatedAtStart,
		CreatedAtEnd:   req.CreatedAtEnd,
	})
	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Order("updated_at desc").Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return
}

type qqCacheListFilter struct {
	QQNum          string
	ClientVersion  string
	DeviceID       string
	ExtractorID    uint
	Extracted      *bool
	CreatedAtStart string
	CreatedAtEnd   string
}

func applyQQCacheListFilters(db *gorm.DB, filter qqCacheListFilter) *gorm.DB {
	if qq := strings.TrimSpace(filter.QQNum); qq != "" {
		db = db.Where("qq_num LIKE ?", "%"+qq+"%")
	}
	if did := strings.TrimSpace(filter.DeviceID); did != "" {
		db = db.Where("device_id LIKE ?", "%"+did+"%")
	}
	if clientVersion := strings.TrimSpace(filter.ClientVersion); clientVersion != "" {
		db = db.Where("client_version LIKE ?", "%"+clientVersion+"%")
	}
	if filter.ExtractorID != 0 {
		db = db.Where("extractor = ?", filter.ExtractorID)
	}
	if filter.Extracted != nil {
		if *filter.Extracted {
			db = db.Where("extractor IS NOT NULL")
		} else {
			db = db.Where("extractor IS NULL")
		}
	}
	return applyQQCacheCreatedAtRangeFilter(db, filter.CreatedAtStart, filter.CreatedAtEnd)
}

func (s *QQCacheService) CountExtractStats() (pending int64, extracted int64, total int64, err error) {
	return s.CountExtractStatsByCreatedRange("", "")
}

func (s *QQCacheService) CountExtractStatsByCreatedRange(createdAtStart string, createdAtEnd string) (pending int64, extracted int64, total int64, err error) {
	db := applyQQCacheCreatedAtRangeFilter(
		global.GVA_DB.Model(&system.SysQQCacheRecord{}),
		createdAtStart,
		createdAtEnd,
	)
	if err = db.Count(&total).Error; err != nil {
		return
	}
	pendingDB := applyQQCacheCreatedAtRangeFilter(
		global.GVA_DB.Model(&system.SysQQCacheRecord{}),
		createdAtStart,
		createdAtEnd,
	).Where("extractor IS NULL")
	if err = pendingDB.Count(&pending).Error; err != nil {
		return
	}
	extractedDB := applyQQCacheCreatedAtRangeFilter(
		global.GVA_DB.Model(&system.SysQQCacheRecord{}),
		createdAtStart,
		createdAtEnd,
	).Where("extractor IS NOT NULL")
	if err = extractedDB.Count(&extracted).Error; err != nil {
		return
	}
	return
}

func (s *QQCacheService) CountBillingSettlementStats() (unsettled int64, settled int64, err error) {
	if err = global.GVA_DB.Model(&system.SysQQCacheRecord{}).
		Where("billing_settled_at IS NULL").
		Count(&unsettled).Error; err != nil {
		return
	}
	if err = global.GVA_DB.Model(&system.SysQQCacheRecord{}).
		Where("billing_settled_at IS NOT NULL").
		Count(&settled).Error; err != nil {
		return
	}
	return
}

func (s *QQCacheService) SettleBilling(operatorRole uint, operatorID uint) (qqCacheBillingSettleResult, error) {
	if operatorRole != qqCacheServiceRoleSuperAdmin && operatorRole != qqCacheServiceRoleAdmin {
		return qqCacheBillingSettleResult{}, errors.New("仅管理员可结算")
	}
	settledAt := time.Now()
	result := qqCacheBillingSettleResult{SettledAt: settledAt}
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		base := tx.Model(&system.SysQQCacheRecord{}).
			Where("created_at <= ? AND billing_settled_at IS NULL", settledAt)
		if err := base.Count(&result.SettledCount).Error; err != nil {
			return err
		}
		if result.SettledCount <= 0 {
			return nil
		}
		return tx.Model(&system.SysQQCacheRecord{}).
			Where("created_at <= ? AND billing_settled_at IS NULL", settledAt).
			Updates(map[string]interface{}{
				"billing_settled_at": settledAt,
				"billing_settled_by": operatorID,
			}).Error
	})
	return result, err
}

func (s *QQCacheService) GetBillingSettlementHistory(operatorRole uint) ([]systemRes.QQCacheBillingSettlementHistoryItem, error) {
	if operatorRole != qqCacheServiceRoleSuperAdmin && operatorRole != qqCacheServiceRoleAdmin {
		return nil, errors.New("仅管理员可查看结算历史")
	}
	var rows []systemRes.QQCacheBillingSettlementHistoryItem
	err := global.GVA_DB.Model(&system.SysQQCacheRecord{}).
		Select("billing_settled_at AS settled_at, COUNT(1) AS settled_count").
		Where("billing_settled_at IS NOT NULL").
		Group("billing_settled_at").
		Order("billing_settled_at DESC").
		Scan(&rows).Error
	return rows, err
}

func applyQQCacheCreatedAtRangeFilter(db *gorm.DB, startRaw string, endRaw string) *gorm.DB {
	if startAt, ok := parseTaskListTime(startRaw); ok {
		db = db.Where("created_at >= ?", startAt)
	}
	if endAt, ok := parseTaskListTime(endRaw); ok {
		db = db.Where("created_at <= ?", endAt)
	}
	return db
}

func (s *QQCacheService) ResetExtractByID(id uint) error {
	if id == 0 {
		return errors.New("记录id不能为空")
	}
	return global.GVA_DB.Model(&system.SysQQCacheRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"extractor":         nil,
			"extract_record_id": nil,
			"extraction_at":     nil,
			"updated_at":        time.Now(),
		}).Error
}

func (s *QQCacheService) ExportPendingIniZipByCount(count int, extractorID uint, createdAtStart string, createdAtEnd string) ([]byte, int, error) {
	if count <= 0 {
		return nil, 0, errors.New("提取数量必须大于0")
	}
	var records []system.SysQQCacheRecord
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		query := applyQQCacheCreatedAtRangeFilter(
			tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("extractor IS NULL").
				Where("ini IS NOT NULL AND TRIM(ini) <> ''"),
			createdAtStart,
			createdAtEnd,
		)
		if err := query.
			Order("updated_at asc").
			Limit(count).
			Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return errors.New("暂无待提取缓存")
		}
		now := time.Now()
		ids := make([]uint, 0, len(records))
		for _, rec := range records {
			ids = append(ids, rec.ID)
			if err := tx.Model(&system.SysQQCacheRecord{}).
				Where("id = ?", rec.ID).
				Where("extractor IS NULL").
				Updates(map[string]any{
					"extractor":         extractorID,
					"extract_record_id": rec.ID,
					"extraction_at":     now,
					"updated_at":        now,
				}).Error; err != nil {
				return err
			}
		}
		return tx.Where("id IN ?", ids).Find(&records).Error
	})
	if err != nil {
		return nil, 0, err
	}
	zipBytes, exportedCount, err := buildQQCacheIniZip(records)
	if err != nil {
		return nil, 0, err
	}
	return zipBytes, exportedCount, nil
}

// ExportIniZipByIDs 将所选记录的 ini 文本打成 zip（zip 内文件名：{qq}.ini）
func (s *QQCacheService) ExportIniZipByIDs(ids []uint) ([]byte, int, error) {
	uniq := make([]uint, 0, len(ids))
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return nil, 0, errors.New("请至少选择一条记录")
	}
	if len(uniq) > qqCacheExportIniMaxIDs {
		return nil, 0, fmt.Errorf("单次最多导出%d条记录", qqCacheExportIniMaxIDs)
	}
	var records []system.SysQQCacheRecord
	if err := global.GVA_DB.Where("id IN ?", uniq).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	if len(records) == 0 {
		return nil, 0, errors.New("未找到所选记录")
	}
	return buildQQCacheIniZip(records)
}

func (s *QQCacheService) ExportIniZipByQQText(raw string) ([]byte, int, error) {
	qqNums := parseQQCacheExportQQNums(raw)
	if len(qqNums) == 0 {
		return nil, 0, errors.New("未解析到QQ账号")
	}
	var records []system.SysQQCacheRecord
	if err := global.GVA_DB.Where("qq_num IN ?", qqNums).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	if len(records) == 0 {
		return nil, 0, errors.New("未找到匹配QQ缓存")
	}
	recordMap := make(map[string]system.SysQQCacheRecord, len(records))
	for _, record := range records {
		recordMap[strings.TrimSpace(record.QQNum)] = record
	}
	ordered := make([]system.SysQQCacheRecord, 0, len(records))
	for _, qqNum := range qqNums {
		if record, ok := recordMap[qqNum]; ok {
			ordered = append(ordered, record)
		}
	}
	if len(ordered) == 0 {
		return nil, 0, errors.New("未找到匹配QQ缓存")
	}
	return buildQQCacheIniZip(ordered)
}

func (s *QQCacheService) ExportAccountListText(req systemReq.QQCacheExportAccountList) (string, int, error) {
	var records []system.SysQQCacheRecord
	ids := uniqueQQCacheExportIDs(req.IDs)
	if len(ids) > 0 {
		if err := global.GVA_DB.Where("id IN ?", ids).Order("id desc").Find(&records).Error; err != nil {
			return "", 0, err
		}
	} else {
		db := applyQQCacheListFilters(global.GVA_DB.Model(&system.SysQQCacheRecord{}), qqCacheListFilter{
			QQNum:          req.QQNum,
			ClientVersion:  req.ClientVersion,
			DeviceID:       req.DeviceID,
			ExtractorID:    req.ExtractorID,
			Extracted:      req.Extracted,
			CreatedAtStart: req.CreatedAtStart,
			CreatedAtEnd:   req.CreatedAtEnd,
		})
		if err := db.Order("updated_at desc").Order("id desc").Find(&records).Error; err != nil {
			return "", 0, err
		}
	}
	if len(records) == 0 {
		return "", 0, errors.New("暂无可导出的账号")
	}
	lines := make([]string, 0, len(records))
	for _, record := range records {
		qqNum := strings.TrimSpace(record.QQNum)
		if qqNum == "" {
			continue
		}
		clientVersion := strings.TrimSpace(record.ClientVersion)
		if clientVersion == "" {
			clientVersion = "-"
		}
		lines = append(lines, fmt.Sprintf("%s----%s", qqNum, clientVersion))
	}
	if len(lines) == 0 {
		return "", 0, errors.New("暂无可导出的账号")
	}
	return strings.Join(lines, "\r\n") + "\r\n", len(lines), nil
}

func uniqueQQCacheExportIDs(ids []uint) []uint {
	uniq := make([]uint, 0, len(ids))
	seen := map[uint]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	return uniq
}

func parseQQCacheExportQQNums(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	seen := map[string]struct{}{}
	qqNums := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		qqNum := strings.TrimSpace(line)
		if idx := strings.Index(qqNum, "----"); idx >= 0 {
			qqNum = strings.TrimSpace(qqNum[:idx])
		}
		if qqNum == "" {
			continue
		}
		valid := true
		for _, r := range qqNum {
			if !unicode.IsDigit(r) {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		if _, ok := seen[qqNum]; ok {
			continue
		}
		seen[qqNum] = struct{}{}
		qqNums = append(qqNums, qqNum)
	}
	return qqNums
}

func buildQQCacheIniZip(records []system.SysQQCacheRecord) ([]byte, int, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	added := 0
	accountLines := make([]string, 0, len(records))
	for _, rec := range records {
		if rec.INI == nil || strings.TrimSpace(*rec.INI) == "" {
			continue
		}
		name := fmt.Sprintf("%s.ini", sanitizeQQCacheZipEntryBase(rec.QQNum))
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		normalizedINI := normalizeQQCacheExportINI(*rec.INI, rec.QQPwd)
		if _, err := w.Write([]byte(normalizedINI)); err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		accountLines = append(accountLines, buildQQCacheAccountLine(rec, normalizedINI))
		added++
	}
	if added > 0 {
		sort.Strings(accountLines)
		w, err := zw.Create("账号.txt")
		if err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
		accountText := strings.Join(accountLines, "\r\n")
		if accountText != "" {
			accountText += "\r\n"
		}
		if _, err := w.Write([]byte(accountText)); err != nil {
			_ = zw.Close()
			return nil, 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	if added == 0 {
		return nil, 0, errors.New("所选记录均无缓存内容可导出")
	}
	return buf.Bytes(), added, nil
}

func buildQQCacheAccountLine(rec system.SysQQCacheRecord, iniText string) string {
	return fmt.Sprintf("%s----%s----%s----%s",
		strings.TrimSpace(rec.QQNum),
		strings.TrimSpace(rec.QQPwd),
		extractQQCacheGUID(iniText),
		formatQQCacheRegisterTime(rec.CreatedAt),
	)
}

func extractQQCacheGUID(raw string) string {
	value := extractQQCacheINIValue(raw, "guid")
	if value == "" {
		return "-"
	}
	return value
}

func extractQQCacheINIValue(raw string, targetKey string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		index := strings.IndexAny(line, "=:")
		if index <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:index])
		if !strings.EqualFold(key, targetKey) {
			continue
		}
		value := strings.TrimSpace(line[index+1:])
		value = strings.Trim(value, `"'`)
		if value != "" {
			return value
		}
	}
	return ""
}

func formatQQCacheRegisterTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func normalizeQQCacheExportINI(raw string, qqPwd string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	output := make([]string, 0, len(lines)+1)
	hasQQPassword := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			output = append(output, line)
			continue
		}
		key, hasKV := splitINIKeyValue(line)
		if !hasKV {
			output = append(output, line)
			continue
		}
		if strings.HasPrefix(key, "_") {
			continue
		}
		if strings.EqualFold(key, "qqpassword") {
			hasQQPassword = true
		}
		if strings.EqualFold(key, "deviceInfo") {
			output = append(output, normalizeQQCacheDeviceInfoLine(key, line))
			continue
		}
		output = append(output, line)
	}

	text := strings.Join(output, "\r\n")
	if strings.TrimSpace(text) == "" {
		return ""
	}
	if !hasQQPassword {
		qqPwd = strings.TrimSpace(qqPwd)
		if qqPwd != "" {
			text = strings.TrimRight(text, "\r\n")
			text += "\r\nqqpassword=" + qqPwd
		}
	}
	if !strings.HasSuffix(text, "\r\n") {
		text += "\r\n"
	}
	return text
}

func normalizeQQCacheDeviceInfoLine(key string, line string) string {
	index := strings.IndexAny(line, "=:")
	if index <= 0 {
		return line
	}
	value := strings.TrimSpace(line[index+1:])
	if value == "" {
		return strings.TrimSpace(key) + string(line[index]) + `{"model":"XiaoMi 17"}`
	}
	var deviceInfo map[string]any
	if err := json.Unmarshal([]byte(value), &deviceInfo); err != nil {
		return line
	}
	deviceInfo["model"] = "XiaoMi 17"
	normalizedValue, err := json.Marshal(deviceInfo)
	if err != nil {
		return line
	}
	return strings.TrimSpace(key) + string(line[index]) + string(normalizedValue)
}

func splitINIKeyValue(line string) (string, bool) {
	index := strings.IndexAny(line, "=:")
	if index <= 0 {
		return "", false
	}
	key := strings.TrimSpace(line[:index])
	if key == "" {
		return "", false
	}
	return key, true
}

func sanitizeQQCacheZipEntryBase(qq string) string {
	qq = strings.TrimSpace(qq)
	var b strings.Builder
	for _, r := range qq {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "qq"
	}
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}

func trimToPtr(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return &raw
}

func stringPtr(raw string) *string {
	v := raw
	return &v
}
