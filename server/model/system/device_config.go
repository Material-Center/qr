package system

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type SysDeviceConfig struct {
	global.GVA_MODEL
	DeviceID    string         `json:"deviceId" gorm:"column:device_id;size:128;uniqueIndex;not null;comment:设备ID"`
	AccountType string         `json:"accountType" gorm:"column:account_type;size:32;not null;default:default;index;comment:账号类型"`
	GroupID     *uint          `json:"groupId" gorm:"column:group_id;index;comment:设备分组ID"`
	Group       SysDeviceGroup `json:"group" gorm:"foreignKey:GroupID;references:ID"`
	Remark      string         `json:"remark" gorm:"column:remark;size:255;comment:备注"`
}

func (SysDeviceConfig) TableName() string {
	return "sys_device_configs"
}
