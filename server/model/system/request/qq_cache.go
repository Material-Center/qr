package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type QQCacheUpload struct {
	TaskID   uint   `json:"taskId" form:"taskId"`
	Phone    string `json:"phone" form:"phone"`
	QQNum    string `json:"qqNum" form:"qqNum"`
	QQPwd    string `json:"qqPwd" form:"qqPwd"`
	INI      string `json:"ini" form:"ini"`
	DeviceID string `json:"deviceId" form:"deviceId"`
}

type InternalToolQQCacheImport struct {
	Phone    string `json:"phone" form:"phone"`
	QQNum    string `json:"qqNum" form:"qqNum"`
	QQPwd    string `json:"qqPwd" form:"qqPwd"`
	INI      string `json:"ini" form:"ini"`
	DeviceID string `json:"deviceId" form:"deviceId"`
	Force    bool   `json:"force" form:"force"`
}

type QQCacheExtract struct {
	QQNum string `json:"qqNum" form:"qqNum"`
}

type QQCacheList struct {
	request.PageInfo
	QQNum          string `json:"qqNum" form:"qqNum"`
	ClientVersion  string `json:"clientVersion" form:"clientVersion"`
	DeviceID       string `json:"deviceId" form:"deviceId"`
	ExtractorID    uint   `json:"extractorId" form:"extractorId"`
	Extracted      *bool  `json:"extracted" form:"extracted"`
	CreatedAtStart string `json:"createdAtStart" form:"createdAtStart"`
	CreatedAtEnd   string `json:"createdAtEnd" form:"createdAtEnd"`
}

type QQCacheResetExtract struct {
	ID uint `json:"id" form:"id"`
}

// QQCacheExportIniZip 管理端批量导出缓存 INI（打包为 zip）
type QQCacheExportIniZip struct {
	IDs []uint `json:"ids" form:"ids"`
}

// QQCacheExportPendingIniZip 管理端按数量提取未提取缓存 INI（打包为 zip）
type QQCacheExportPendingIniZip struct {
	Count          int    `json:"count" form:"count"`
	CreatedAtStart string `json:"createdAtStart" form:"createdAtStart"`
	CreatedAtEnd   string `json:"createdAtEnd" form:"createdAtEnd"`
}

type QQCacheSalesExtract struct {
	Count int `json:"count" form:"count"`
}

type QQCacheSalesHistory struct {
	request.PageInfo
	Date string `json:"date" form:"date"`
}

type QQCacheSalesSettle struct {
	ExtractorID uint `json:"extractorId" form:"extractorId"`
}

type QQCacheExportAccountList struct {
	IDs            []uint `json:"ids" form:"ids"`
	QQNum          string `json:"qqNum" form:"qqNum"`
	ClientVersion  string `json:"clientVersion" form:"clientVersion"`
	DeviceID       string `json:"deviceId" form:"deviceId"`
	ExtractorID    uint   `json:"extractorId" form:"extractorId"`
	Extracted      *bool  `json:"extracted" form:"extracted"`
	CreatedAtStart string `json:"createdAtStart" form:"createdAtStart"`
	CreatedAtEnd   string `json:"createdAtEnd" form:"createdAtEnd"`
}
