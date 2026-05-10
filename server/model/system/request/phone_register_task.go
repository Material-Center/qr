package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type PhoneRegisterTaskCreate struct {
	Phone          string `json:"phone" form:"phone"`
	SMSReceiveMode string `json:"smsReceiveMode" form:"smsReceiveMode"`
}

type PhoneRegisterTaskSubmitCode struct {
	TaskID     uint   `json:"taskId" form:"taskId"`
	VerifyCode string `json:"verifyCode" form:"verifyCode"`
}

type PhoneRegisterTaskList struct {
	request.PageInfo
	PromoterID      uint   `json:"promoterId" form:"promoterId"`
	LeaderID        uint   `json:"leaderId" form:"leaderId"`
	Status          string `json:"status" form:"status"`
	StatusCode      *int   `json:"statusCode" form:"statusCode"`
	Phone           string `json:"phone" form:"phone"`
	QQNum           string `json:"qqNum" form:"qqNum"`
	DeviceID        string `json:"deviceId" form:"deviceId"`
	SMSReceiveMode  string `json:"smsReceiveMode" form:"smsReceiveMode"`
	FinishedAtStart string `json:"finishedAtStart" form:"finishedAtStart"`
	FinishedAtEnd   string `json:"finishedAtEnd" form:"finishedAtEnd"`
	DayScoped       bool   `json:"dayScoped" form:"dayScoped"`
}

type PhoneRegisterTaskSummaryFilter struct {
	LeaderID        uint   `json:"leaderId" form:"leaderId"`
	FinishedAtStart string `json:"finishedAtStart" form:"finishedAtStart"`
	FinishedAtEnd   string `json:"finishedAtEnd" form:"finishedAtEnd"`
	DayScoped       bool   `json:"dayScoped" form:"dayScoped"`
}

type PhoneRegisterTaskSettle struct {
	LeaderID uint `json:"leaderId" form:"leaderId"`
}

type PhoneRegisterTaskSettlementHistory struct {
	LeaderID uint `json:"leaderId" form:"leaderId"`
}

type PhoneRegisterDevicePoll struct {
	DeviceID string `json:"deviceId" form:"deviceId"`
}

type PhoneRegisterDeviceTask struct {
	DeviceID string `json:"deviceId" form:"deviceId"`
}

type PhoneRegisterDeviceHeartbeat struct {
	DeviceID string `json:"deviceId" form:"deviceId"`
}

type PhoneRegisterDeviceReport struct {
	DeviceID   string `json:"deviceId" form:"deviceId"`
	Action     string `json:"action" form:"action"`
	Message    string `json:"message" form:"message"`
	StatusCode *int   `json:"statusCode" form:"statusCode"`
}

type PhoneRegisterDeviceLog struct {
	DeviceID   string `json:"deviceId" form:"deviceId"`
	TaskID     uint   `json:"taskId" form:"taskId"`
	ClientTime string `json:"clientTime" form:"clientTime"`
	Message    string `json:"message" form:"message"`
}

type PhoneRegisterTaskLogList struct {
	request.PageInfo
	TaskID uint `json:"taskId" form:"taskId"`
}
