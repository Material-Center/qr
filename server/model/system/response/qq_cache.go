package response

import "github.com/flipped-aurora/gin-vue-admin/server/model/system"

type QQCacheListResponse struct {
	List     []system.SysQQCacheRecord `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}
