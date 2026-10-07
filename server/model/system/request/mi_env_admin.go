package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

// MIEnvAdminList is shared by the admin list and filtered delete-all actions.
// Empty filter fields intentionally mean "all active records".
type MIEnvAdminList struct {
	request.PageInfo
	UpdatedAtStart string `json:"updatedAtStart" form:"updatedAtStart"`
	UpdatedAtEnd   string `json:"updatedAtEnd" form:"updatedAtEnd"`
	DeviceID       string `json:"deviceId" form:"deviceId"`
	GroupID        *uint  `json:"groupId" form:"groupId"`
	Ungrouped      bool   `json:"ungrouped" form:"ungrouped"`
	Type           string `json:"type" form:"type"`
	MinUsage       *int   `json:"minUsage" form:"minUsage"`
	MaxUsage       *int   `json:"maxUsage" form:"maxUsage"`
	MinMadeCount   *int   `json:"minMadeCount" form:"minMadeCount"`
	MaxMadeCount   *int   `json:"maxMadeCount" form:"maxMadeCount"`
	Status         string `json:"status" form:"status"`
	Confirm        bool   `json:"confirm" form:"confirm"`
}

type MIEnvAdminDeleteSelected struct {
	IDs     []uint `json:"ids"`
	Confirm bool   `json:"confirm"`
}
