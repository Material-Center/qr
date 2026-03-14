package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type RegisterTaskCreate struct {
	Phone string `json:"phone" form:"phone"`
}

type RegisterTaskStep struct {
	TaskID      uint   `json:"taskId" form:"taskId"`
	Step        string `json:"step" form:"step"`
	VerifyCode  string `json:"verifyCode" form:"verifyCode"`
	Action      string `json:"action" form:"action"` // submit/retry/fail
	FailMessage string `json:"failMessage" form:"failMessage"`
}

type RegisterTaskList struct {
	request.PageInfo
	PromoterID uint  `json:"promoterId" form:"promoterId"`
	LeaderID   uint  `json:"leaderId" form:"leaderId"`
	StatusCode *int  `json:"statusCode" form:"statusCode"`
	Unfinished *bool `json:"unfinished" form:"unfinished"`
}

type RegisterTaskSummaryFilter struct {
	LeaderID uint `json:"leaderId" form:"leaderId"`
}
