package response

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

type MIEnvAdminItem struct {
	ID                uint       `json:"id"`
	DeviceID          string     `json:"deviceId"`
	Type              string     `json:"type"`
	SerialBackupName  string     `json:"serialBackupName"`
	AndroidID         string     `json:"androidId"`
	UsageCount        int        `json:"usageCount"`
	MaxUsage          int        `json:"maxUsage"`
	MadeCount         int        `json:"madeCount"`
	Frozen            bool       `json:"frozen"`
	ConsumedAt        *time.Time `json:"consumedAt"`
	LastUsedAt        *time.Time `json:"lastUsedAt"`
	MakeReservedUntil *time.Time `json:"makeReservedUntil"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func NewMIEnvAdminItems(records []system.SysMIEnvRecord) []MIEnvAdminItem {
	items := make([]MIEnvAdminItem, 0, len(records))
	for _, record := range records {
		items = append(items, MIEnvAdminItem{
			ID:                record.ID,
			DeviceID:          record.DeviceID,
			Type:              record.Type,
			SerialBackupName:  record.SerialBackupName,
			AndroidID:         record.AndroidID,
			UsageCount:        record.UsageCount,
			MaxUsage:          record.MaxUsage,
			MadeCount:         record.MadeCount,
			Frozen:            record.Frozen,
			ConsumedAt:        record.ConsumedAt,
			LastUsedAt:        record.LastUsedAt,
			MakeReservedUntil: record.MakeReservedUntil,
			CreatedAt:         record.CreatedAt,
			UpdatedAt:         record.UpdatedAt,
		})
	}
	return items
}
