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
