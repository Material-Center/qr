package domain

import "time"

type TaskType string
type SourceType string
type APIType string
type HTTPMethod string
type ResponseType string
type JobStatus string
type JobItemStatus string

const (
	TaskTypeSendCode    TaskType = "send_code"
	TaskTypeReceiveCode TaskType = "receive_code"

	SourceTypeAPI  SourceType = "api"
	SourceTypeTXT  SourceType = "txt"
	SourceTypeNone SourceType = "none"

	APITypePhoneSource APIType = "phone_source"
	APITypeCodeSource  APIType = "code_source"

	HTTPMethodGET  HTTPMethod = "GET"
	HTTPMethodPOST HTTPMethod = "POST"

	ResponseTypeAuto ResponseType = "auto"
	ResponseTypeJSON ResponseType = "json"
	ResponseTypeText ResponseType = "text"

	JobStatusPending  JobStatus = "pending"
	JobStatusRunning  JobStatus = "running"
	JobStatusPaused   JobStatus = "paused"
	JobStatusStopped  JobStatus = "stopped"
	JobStatusFinished JobStatus = "finished"

	JobItemStatusPending       JobItemStatus = "pending"
	JobItemStatusCreated       JobItemStatus = "created"
	JobItemStatusWaitingCode   JobItemStatus = "waiting_code"
	JobItemStatusCodeSubmitted JobItemStatus = "code_submitted"
	JobItemStatusSucceeded     JobItemStatus = "succeeded"
	JobItemStatusFailed        JobItemStatus = "failed"
	JobItemStatusStopped       JobItemStatus = "stopped"
)

type Profile struct {
	ID              int64
	Name            string
	TokenRef        string
	TokenMask       string
	BaseURLOverride string
	CreateDelay     time.Duration
	Enabled         bool
	Remark          string
}

type APITemplate struct {
	ID           int64
	Name         string
	APIType      APIType
	Method       HTTPMethod
	URL          string
	Headers      map[string]string
	Query        map[string]string
	BodyTemplate string
	ResponseType ResponseType
	ExtractRule  map[string]string
	NotReadyRule map[string]string
	SuccessRule  map[string]string
	ErrorRule    map[string]string
	Enabled      bool
	Remark       string
}

type TaskTemplate struct {
	ID                 int64
	Name               string
	ProfileID          int64
	TaskType           TaskType
	PhoneSourceType    SourceType
	PhoneAPITemplateID int64
	DefaultTXTDir      string
	CodeSourceType     SourceType
	CodeAPITemplateID  int64
	FailedOutputDir    string
	Enabled            bool
	Remark             string
}

type DevicePoolSnapshot struct {
	BaseURL             string
	IdleDeviceCount     int64
	ReserveDevices      int64
	Capacity            int64
	RunningProfileCount int
	RunningJobCount     int
	QueryElapsed        time.Duration
	LastError           string
	CreatedAt           time.Time
}

type GlobalSettings struct {
	BaseURL        string
	ReserveDevices int64
	Interval       time.Duration
	Timeout        time.Duration
	LogDir         string
}

type Job struct {
	ID                       int64
	ProfileID                int64
	TaskTemplateID           int64
	Name                     string
	TaskType                 TaskType
	PhoneSourceType          SourceType
	CodeSourceType           SourceType
	PhoneSourceConfigJSON    string
	CodeSourceConfigJSON     string
	APITemplateSnapshotJSON  string
	TaskTemplateSnapshotJSON string
	BaseURLSnapshot          string
	ReserveDevicesSnapshot   int64
	IntervalSnapshot         time.Duration
	TimeoutSnapshot          time.Duration
	CreateDelaySnapshot      time.Duration
	Status                   JobStatus
	Paused                   bool
	Stopped                  bool
	CreatedAt                time.Time
	StartedAt                *time.Time
	FinishedAt               *time.Time
	UpdatedAt                time.Time
}

type JobItem struct {
	ID           int64
	JobID        int64
	Phone        string
	RemoteTaskID uint
	Status       JobItemStatus
	RemoteStatus string
	VerifyCode   string
	LastError    string
	Attempts     int
	SourceLineNo int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Event struct {
	ID         int64
	JobID      int64
	ItemID     int64
	Phone      string
	Level      string
	EventType  string
	Message    string
	DetailJSON string
	CreatedAt  time.Time
}
