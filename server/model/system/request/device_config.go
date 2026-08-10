package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type DeviceConfigList struct {
	request.PageInfo
	DeviceID    string `json:"deviceId" form:"deviceId"`
	AccountType string `json:"accountType" form:"accountType"`
}

type DeviceConfigSave struct {
	ID          uint   `json:"id" form:"id"`
	DeviceID    string `json:"deviceId" form:"deviceId"`
	AccountType string `json:"accountType" form:"accountType"`
	Remark      string `json:"remark" form:"remark"`
}

type DeviceConfigDelete struct {
	ID uint `json:"id" form:"id"`
}
