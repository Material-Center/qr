package response

import "time"

type PhoneRegisterTaskActiveInfo struct {
	ID                  uint       `json:"id"`
	CreatedAt           time.Time  `json:"createdAt"`
	Phone               string     `json:"phone"`
	SMSReceiveMode      string     `json:"smsReceiveMode"`
	TaskSource          string     `json:"taskSource,omitempty"`
	CacheStatus         string     `json:"cacheStatus,omitempty"`
	Status              string     `json:"status"`
	StatusCode          *int       `json:"statusCode"`
	LastError           string     `json:"lastError"`
	NeedPromoterCode    bool       `json:"needPromoterCode"`
	CodeSubmitExpiresAt *time.Time `json:"codeSubmitExpiresAt,omitempty"`
	HolderDeviceID      *string    `json:"holderDeviceId,omitempty"`
	ClaimedAt           *time.Time `json:"claimedAt,omitempty"`
	LastHeartbeatAt     *time.Time `json:"lastHeartbeatAt,omitempty"`
	ExpiresAt           time.Time  `json:"expiresAt"`
	FinishedAt          *time.Time `json:"finishedAt"`
}

type PhoneRegisterSubmitStatusResponse struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

type PhoneRegisterTaskListResponse struct {
	List              interface{} `json:"list"`
	Total             int64       `json:"total"`
	Page              int         `json:"page"`
	PageSize          int         `json:"pageSize"`
	SuccessCount      int64       `json:"successCount"`
	FailCount         int64       `json:"failCount"`
	ProcessingCount   int64       `json:"processingCount"`
	DeviceOnlineCount int64       `json:"deviceOnlineCount"`
	DeviceIdleCount   int64       `json:"deviceIdleCount"`
}

type PhoneRegisterTaskUserBrief struct {
	ID       uint   `json:"ID"`
	UserName string `json:"userName"`
	NickName string `json:"nickName"`
}

type PhoneRegisterTaskListItem struct {
	ID              uint                        `json:"ID"`
	CreatedAt       time.Time                   `json:"CreatedAt"`
	Phone           string                      `json:"phone"`
	SMSReceiveMode  string                      `json:"smsReceiveMode"`
	TaskSource      string                      `json:"taskSource,omitempty"`
	CacheStatus     string                      `json:"cacheStatus,omitempty"`
	QQNum           string                      `json:"qqNum,omitempty"`
	Status          string                      `json:"status"`
	StatusCode      *int                        `json:"statusCode"`
	LastError       string                      `json:"lastError"`
	FinishedAt      *time.Time                  `json:"finishedAt"`
	SettledAt       *time.Time                  `json:"settledAt"`
	HolderDeviceID  *string                     `json:"holderDeviceId,omitempty"`
	ClaimedAt       *time.Time                  `json:"claimedAt,omitempty"`
	LastHeartbeatAt *time.Time                  `json:"lastHeartbeatAt,omitempty"`
	ExpiresAt       time.Time                   `json:"expiresAt"`
	Promoter        *PhoneRegisterTaskUserBrief `json:"promoter,omitempty"`
	Leader          *PhoneRegisterTaskUserBrief `json:"leader,omitempty"`
}

type PhoneRegisterTaskLogListResponse struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

type PhoneRegisterTaskSummaryItem struct {
	LeaderID        uint   `json:"leaderId"`
	LeaderName      string `json:"leaderName"`
	PromoterID      uint   `json:"promoterId"`
	PromoterName    string `json:"promoterName"`
	SuccessCount    int64  `json:"successCount"`
	FailCount       int64  `json:"failCount"`
	RiskFailCount   *int64 `json:"riskFailCount,omitempty"`
	ProcessingCount int64  `json:"processingCount"`
	SettledCount    int64  `json:"settledCount"`
	UnsettledCount  int64  `json:"unsettledCount"`
}

type PhoneRegisterTaskSummaryResponse struct {
	Leaders   []PhoneRegisterTaskSummaryItem `json:"leaders"`
	Promoters []PhoneRegisterTaskSummaryItem `json:"promoters"`
}

type PhoneRegisterTaskSettlementHistoryItem struct {
	SettledAt    time.Time `json:"settledAt"`
	SettledCount int64     `json:"settledCount"`
}

type PhoneRegisterDeviceTaskInfo struct {
	TaskID           uint       `json:"taskId"`
	Phone            string     `json:"phone,omitempty"`
	SMSReceiveMode   string     `json:"smsReceiveMode,omitempty"`
	TaskSource       string     `json:"taskSource,omitempty"`
	CacheStatus      string     `json:"cacheStatus,omitempty"`
	Status           string     `json:"status,omitempty"`
	NeedPromoterCode bool       `json:"needPromoterCode"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
	ClaimedAt        *time.Time `json:"claimedAt,omitempty"`
	LastHeartbeatAt  *time.Time `json:"lastHeartbeatAt,omitempty"`
	VerifyCode       string     `json:"verifyCode,omitempty"`
}

type PhoneRegisterOpenAPITaskInfo struct {
	TaskID      uint       `json:"taskId"`
	Phone       string     `json:"phone,omitempty"`
	VerifyMode  string     `json:"verifyMode,omitempty"`
	TaskSource  string     `json:"taskSource,omitempty"`
	CacheStatus string     `json:"cacheStatus,omitempty"`
	Status      string     `json:"status,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	NeedCode    bool       `json:"needCode"`
	HasTask     bool       `json:"hasTask"`
}

type PhoneRegisterOpenAPIVerifyCodeResponse struct {
	TaskID     uint   `json:"taskId"`
	VerifyCode string `json:"verifyCode"`
	HasCode    bool   `json:"hasCode"`
}

type PhoneRegisterOpenAPIReportResponse struct {
	OK              bool   `json:"ok"`
	TaskID          uint   `json:"taskId,omitempty"`
	QQCacheRecordID uint   `json:"qqCacheRecordId,omitempty"`
	QQNum           string `json:"qqNum,omitempty"`
}

type PhoneRegisterDeviceHeartbeatResponse struct {
	OK bool `json:"ok"`
}

type PhoneRegisterImageVerifyConfig struct {
	Provider  string `json:"provider"`
	Endpoint  string `json:"endpoint"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ModelName string `json:"modelName"`
	KeyCode   string `json:"keyCode"`
	RequestID string `json:"requestId"`
	Version   string `json:"version"`
	Question  string `json:"question"`
	System    string `json:"system"`
}

type PhoneRegisterDeviceConfigResponse struct {
	ImageVerify PhoneRegisterImageVerifyConfig `json:"imageVerify"`
}

type QQCacheUploadPhoneRegisterResponse struct {
	QQCacheRecordID uint   `json:"qqCacheRecordId"`
	QQNum           string `json:"qqNum"`
}
