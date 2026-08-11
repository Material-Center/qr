package system

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type SysDeviceGroup struct {
	global.GVA_MODEL
	Name   string `json:"name" gorm:"column:name;size:64;uniqueIndex;not null;comment:分组名称"`
	Remark string `json:"remark" gorm:"column:remark;size:255;comment:备注"`
}

func (SysDeviceGroup) TableName() string {
	return "sys_device_groups"
}
