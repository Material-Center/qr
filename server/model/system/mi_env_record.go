package system

import "time"

// SysMIEnvRecord stores the device environment pool used by the MI client.
// DeletedAt is managed explicitly instead of GORM soft-delete semantics so
// compatibility stats can still report deleted records.
type SysMIEnvRecord struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	DeviceCode       string     `json:"deviceCode" gorm:"column:device_code;size:128;not null;index"`
	DeviceID         string     `json:"deviceId" gorm:"column:device_id;size:128;not null;index"`
	Type             string     `json:"type" gorm:"column:type;size:64;not null;index"`
	SerialBackupName string     `json:"serialBackupName" gorm:"column:serial_backup_name;size:255;not null;index"`
	AndroidID        string     `json:"androidId" gorm:"column:android_id;size:255;not null"`
	Key              string     `json:"key" gorm:"column:env_key;size:255;not null"`
	UsageCount       int        `json:"usageCount" gorm:"column:usage_count;not null;default:0"`
	MaxUsage         int        `json:"maxUsage" gorm:"column:max_usage;not null;default:1"`
	MadeCount        int        `json:"madeCount" gorm:"column:made_count;not null;default:1"`
	Frozen           bool       `json:"frozen" gorm:"column:frozen;not null;default:false;index"`
	ConsumedAt       *time.Time `json:"consumedAt" gorm:"column:consumed_at"`
	LastUsedAt       *time.Time `json:"lastUsedAt" gorm:"column:last_used_at"`
	CreatedAt        time.Time  `json:"createdAt" gorm:"column:created_at;not null;index"`
	UpdatedAt        time.Time  `json:"updatedAt" gorm:"column:updated_at;not null"`
	DeletedAt        *time.Time `json:"deletedAt" gorm:"column:deleted_at;index"`
}

func (SysMIEnvRecord) TableName() string { return "sys_mi_env_records" }
