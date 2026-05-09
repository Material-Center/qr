package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type QQCacheUpload struct {
	Phone    string `json:"phone" form:"phone"`
	QQNum    string `json:"qqNum" form:"qqNum"`
	QQPwd    string `json:"qqPwd" form:"qqPwd"`
	INI      string `json:"ini" form:"ini"`
	DeviceID string `json:"deviceId" form:"deviceId"`
}

type QQCacheExtract struct {
	QQNum string `json:"qqNum" form:"qqNum"`
}

type QQCacheList struct {
	request.PageInfo
	QQNum       string `json:"qqNum" form:"qqNum"`
	DeviceID    string `json:"deviceId" form:"deviceId"`
	ExtractorID uint   `json:"extractorId" form:"extractorId"`
	Extracted   *bool  `json:"extracted" form:"extracted"`
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
	Count int `json:"count" form:"count"`
}
