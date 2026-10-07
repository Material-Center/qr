package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/gorm"
)

// SysMIEnvRecord stores the device environment pool used by the MI client.
type SysMIEnvRecord struct {
	global.GVA_MODEL
	DeviceCode        string     `json:"deviceCode" gorm:"column:device_code;size:128;not null;index"`
	DeviceID          string     `json:"deviceId" gorm:"column:device_id;size:128;not null;index"`
	Type              string     `json:"type" gorm:"column:type;size:64;not null;index"`
	SerialBackupName  string     `json:"serialBackupName" gorm:"column:serial_backup_name;size:255;not null;index"`
	AndroidID         string     `json:"androidId" gorm:"column:android_id;size:255;not null"`
	Key               string     `json:"key" gorm:"column:env_key;size:255;not null"`
	UsageCount        int        `json:"usageCount" gorm:"column:usage_count;not null;default:0;index"`
	MaxUsage          int        `json:"maxUsage" gorm:"column:max_usage;not null;default:1"`
	MadeCount         int        `json:"madeCount" gorm:"column:made_count;not null;default:1;index"`
	Frozen            bool       `json:"frozen" gorm:"column:frozen;not null;default:false;index"`
	ConsumedAt        *time.Time `json:"consumedAt" gorm:"column:consumed_at"`
	LastUsedAt        *time.Time `json:"lastUsedAt" gorm:"column:last_used_at"`
	MakeReservedUntil *time.Time `json:"makeReservedUntil" gorm:"column:make_reserved_until;index"`
}

func (SysMIEnvRecord) TableName() string { return "sys_mi_env_records" }

// miEnvRecordIndexSchema keeps the large-table management indexes separate
// from GVA_MODEL. The shared base model owns the lifecycle columns, while this
// migration-only schema describes indexes that combine those columns with
// environment-specific fields.
type miEnvRecordIndexSchema struct {
	ID         uint           `gorm:"column:id;index:idx_mi_env_admin_active_updated,priority:3"`
	DeviceID   string         `gorm:"column:device_id;index:idx_mi_env_admin_device_active_type,priority:1"`
	Type       string         `gorm:"column:type;index:idx_mi_env_admin_device_active_type,priority:3"`
	UsageCount int            `gorm:"column:usage_count;index:idx_mi_env_admin_active_usage,priority:2"`
	MadeCount  int            `gorm:"column:made_count;index:idx_mi_env_admin_active_made,priority:2"`
	Frozen     bool           `gorm:"column:frozen;index:idx_mi_env_admin_active_frozen_updated,priority:2"`
	CreatedAt  time.Time      `gorm:"column:created_at;index:idx_sys_mi_env_records_created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;index:idx_sys_mi_env_records_updated_at;index:idx_mi_env_admin_active_updated,priority:2;index:idx_mi_env_admin_active_frozen_updated,priority:3"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index:idx_mi_env_admin_active_updated,priority:1;index:idx_mi_env_admin_device_active_type,priority:2;index:idx_mi_env_admin_active_usage,priority:1;index:idx_mi_env_admin_active_made,priority:1;index:idx_mi_env_admin_active_frozen_updated,priority:1"`
}

func (miEnvRecordIndexSchema) TableName() string { return SysMIEnvRecord{}.TableName() }

// EnsureMIEnvRecordIndexes creates the management-page composite indexes
// without duplicating GVA_MODEL fields on SysMIEnvRecord.
func EnsureMIEnvRecordIndexes(db *gorm.DB) error {
	for _, name := range []string{
		"idx_sys_mi_env_records_created_at",
		"idx_sys_mi_env_records_updated_at",
		"idx_mi_env_admin_active_updated",
		"idx_mi_env_admin_device_active_type",
		"idx_mi_env_admin_active_usage",
		"idx_mi_env_admin_active_made",
		"idx_mi_env_admin_active_frozen_updated",
	} {
		if db.Migrator().HasIndex(&miEnvRecordIndexSchema{}, name) {
			continue
		}
		if err := db.Migrator().CreateIndex(&miEnvRecordIndexSchema{}, name); err != nil {
			return err
		}
	}
	return nil
}
