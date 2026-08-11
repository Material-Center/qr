package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type DeviceConfigList struct {
	request.PageInfo
	DeviceID    string `json:"deviceId" form:"deviceId"`
	AccountType string `json:"accountType" form:"accountType"`
	GroupID     *uint  `json:"groupId" form:"groupId"`
	Ungrouped   bool   `json:"ungrouped" form:"ungrouped"`
}

type DeviceConfigSave struct {
	ID          uint   `json:"id" form:"id"`
	DeviceID    string `json:"deviceId" form:"deviceId"`
	AccountType string `json:"accountType" form:"accountType"`
	GroupID     *uint  `json:"groupId" form:"groupId"`
	Remark      string `json:"remark" form:"remark"`
}

type DeviceConfigDelete struct {
	ID uint `json:"id" form:"id"`
}

type DeviceGroupSave struct {
	ID     uint   `json:"id" form:"id"`
	Name   string `json:"name" form:"name"`
	Remark string `json:"remark" form:"remark"`
}

type DeviceGroupDelete struct {
	ID uint `json:"id" form:"id"`
}
