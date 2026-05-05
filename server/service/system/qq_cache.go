package system

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const qqCacheExportIniMaxIDs = 100

type QQCacheService struct{}

var QQCacheServiceApp = new(QQCacheService)

func (s *QQCacheService) UploadByApp(userID uint, req systemReq.QQCacheUpload) (system.SysQQCacheRecord, error) {
	_ = userID
	return s.uploadRecord(req)
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
		completedTask, taskErr := (&PhoneRegisterTaskService{}).CompleteTaskAfterQQCacheUploadTx(tx, strings.TrimSpace(req.DeviceID), record.ID, record.QQNum)
		if taskErr != nil {
			return taskErr
		}
		task = completedTask
		return nil
	})
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
	entity := system.SysQQCacheRecord{
		Phone:    phone,
		QQNum:    qqNum,
		QQPwd:    strings.TrimSpace(req.QQPwd),
		INI:      stringPtr(iniText),
		DeviceID: deviceID,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "qq_num"}},
		DoUpdates: clause.Assignments(map[string]any{
			"phone":             phone,
			"qq_pwd":            strings.TrimSpace(req.QQPwd),
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
	db := global.GVA_DB.Model(&system.SysQQCacheRecord{})
	if qq := strings.TrimSpace(req.QQNum); qq != "" {
		db = db.Where("qq_num LIKE ?", "%"+qq+"%")
	}
	if did := strings.TrimSpace(req.DeviceID); did != "" {
		db = db.Where("device_id LIKE ?", "%"+did+"%")
	}
	if req.ExtractorID != 0 {
		db = db.Where("extractor = ?", req.ExtractorID)
	}
	if req.Extracted != nil {
		if *req.Extracted {
			db = db.Where("extractor IS NOT NULL")
		} else {
			db = db.Where("extractor IS NULL")
		}
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Order("updated_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return
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

// ExportIniZipByIDs 将所选记录的 ini 文本打成 zip（zip 内文件名：{id}_{qq}.ini）
func (s *QQCacheService) ExportIniZipByIDs(ids []uint) ([]byte, error) {
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
		return nil, errors.New("请至少选择一条记录")
	}
	if len(uniq) > qqCacheExportIniMaxIDs {
		return nil, fmt.Errorf("单次最多导出%d条记录", qqCacheExportIniMaxIDs)
	}
	var records []system.SysQQCacheRecord
	if err := global.GVA_DB.Where("id IN ?", uniq).Find(&records).Error; err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("未找到所选记录")
	}
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	added := 0
	for _, rec := range records {
		if rec.INI == nil || strings.TrimSpace(*rec.INI) == "" {
			continue
		}
		name := fmt.Sprintf("%s.ini", sanitizeQQCacheZipEntryBase(rec.QQNum))
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		normalizedINI := normalizeQQCacheExportINI(*rec.INI)
		if _, err := w.Write([]byte(normalizedINI)); err != nil {
			_ = zw.Close()
			return nil, err
		}
		added++
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if added == 0 {
		return nil, errors.New("所选记录均无缓存内容可导出")
	}
	return buf.Bytes(), nil
}

func normalizeQQCacheExportINI(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	output := make([]string, 0, len(lines))

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
