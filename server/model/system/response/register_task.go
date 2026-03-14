package response

import (
	"time"
)

type RegisterTaskSummaryItem struct {
	LeaderID        uint   `json:"leaderId"`
	LeaderName      string `json:"leaderName"`
	PromoterID      uint   `json:"promoterId"`
	PromoterName    string `json:"promoterName"`
	SuccessCount    int64  `json:"successCount"`
	FailCount       int64  `json:"failCount"`
	ProcessingCount int64  `json:"processingCount"`
}

type RegisterTaskSummaryResponse struct {
	Leaders   []RegisterTaskSummaryItem `json:"leaders"`
	Promoters []RegisterTaskSummaryItem `json:"promoters"`
}

type RegisterTaskActiveInfo struct {
	ID          uint       `json:"id"`
	Phone       string     `json:"phone"`
	CurrentStep string     `json:"currentStep"`
	StatusCode  *int       `json:"statusCode"`
	LastError   string     `json:"lastError"`
	RetryCount  int        `json:"retryCount"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	FinishedAt  *time.Time `json:"finishedAt"`
}

type RegisterTaskListResponse struct {
	List            interface{} `json:"list"`
	Total           int64       `json:"total"`
	Page            int         `json:"page"`
	PageSize        int         `json:"pageSize"`
	SuccessCount    int64       `json:"successCount"`
	FailCount       int64       `json:"failCount"`
	ProcessingCount int64       `json:"processingCount"`
}
