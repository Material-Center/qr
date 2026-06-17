package store

import (
	"strconv"
	"time"

	"phone-task-client/internal/domain"
)

type settingModel struct {
	Key       string `gorm:"primaryKey"`
	Value     string
	UpdatedAt time.Time
}

type profileModel struct {
	ID              int64 `gorm:"primaryKey;autoIncrement"`
	Name            string
	TokenRef        string
	TokenMask       string
	BaseURLOverride string
	CreateDelayMS   int64
	Enabled         bool
	Remark          string
}

func profileModelFromDomain(v domain.Profile) profileModel {
	return profileModel{
		ID:              v.ID,
		Name:            v.Name,
		TokenRef:        v.TokenRef,
		TokenMask:       v.TokenMask,
		BaseURLOverride: v.BaseURLOverride,
		CreateDelayMS:   v.CreateDelay.Milliseconds(),
		Enabled:         v.Enabled,
		Remark:          v.Remark,
	}
}

func (m profileModel) toDomain() domain.Profile {
	return domain.Profile{
		ID:              m.ID,
		Name:            m.Name,
		TokenRef:        m.TokenRef,
		TokenMask:       m.TokenMask,
		BaseURLOverride: m.BaseURLOverride,
		CreateDelay:     time.Duration(m.CreateDelayMS) * time.Millisecond,
		Enabled:         m.Enabled,
		Remark:          m.Remark,
	}
}

type apiTemplateModel struct {
	ID               int64 `gorm:"primaryKey;autoIncrement"`
	Name             string
	APIType          string
	Method           string
	URL              string
	HeadersJSON      string
	QueryJSON        string
	BodyTemplate     string
	ResponseType     string
	ExtractRuleJSON  string
	NotReadyRuleJSON string
	SuccessRuleJSON  string
	ErrorRuleJSON    string
	Enabled          bool
	Remark           string
}

func apiTemplateModelFromDomain(v domain.APITemplate) (apiTemplateModel, error) {
	return apiTemplateModel{
		ID:               v.ID,
		Name:             v.Name,
		APIType:          string(v.APIType),
		Method:           string(v.Method),
		URL:              v.URL,
		HeadersJSON:      mustJSON(v.Headers),
		QueryJSON:        mustJSON(v.Query),
		BodyTemplate:     v.BodyTemplate,
		ResponseType:     string(v.ResponseType),
		ExtractRuleJSON:  mustJSON(v.ExtractRule),
		NotReadyRuleJSON: mustJSON(v.NotReadyRule),
		SuccessRuleJSON:  mustJSON(v.SuccessRule),
		ErrorRuleJSON:    mustJSON(v.ErrorRule),
		Enabled:          v.Enabled,
		Remark:           v.Remark,
	}, nil
}

func (m apiTemplateModel) toDomain() (domain.APITemplate, error) {
	return domain.APITemplate{
		ID:           m.ID,
		Name:         m.Name,
		APIType:      domain.APIType(m.APIType),
		Method:       domain.HTTPMethod(m.Method),
		URL:          m.URL,
		Headers:      parseJSONMap(m.HeadersJSON),
		Query:        parseJSONMap(m.QueryJSON),
		BodyTemplate: m.BodyTemplate,
		ResponseType: domain.ResponseType(m.ResponseType),
		ExtractRule:  parseJSONMap(m.ExtractRuleJSON),
		NotReadyRule: parseJSONMap(m.NotReadyRuleJSON),
		SuccessRule:  parseJSONMap(m.SuccessRuleJSON),
		ErrorRule:    parseJSONMap(m.ErrorRuleJSON),
		Enabled:      m.Enabled,
		Remark:       m.Remark,
	}, nil
}

type taskTemplateModel struct {
	ID                 int64 `gorm:"primaryKey;autoIncrement"`
	Name               string
	ProfileID          int64
	TaskType           string
	PhoneSourceType    string
	PhoneAPITemplateID int64
	DefaultTXTDir      string
	CodeSourceType     string
	CodeAPITemplateID  int64
	FailedOutputDir    string
	Enabled            bool
	Remark             string
}

func taskTemplateModelFromDomain(v domain.TaskTemplate) taskTemplateModel {
	return taskTemplateModel{
		ID:                 v.ID,
		Name:               v.Name,
		ProfileID:          v.ProfileID,
		TaskType:           string(v.TaskType),
		PhoneSourceType:    string(v.PhoneSourceType),
		PhoneAPITemplateID: v.PhoneAPITemplateID,
		DefaultTXTDir:      v.DefaultTXTDir,
		CodeSourceType:     string(v.CodeSourceType),
		CodeAPITemplateID:  v.CodeAPITemplateID,
		FailedOutputDir:    v.FailedOutputDir,
		Enabled:            v.Enabled,
		Remark:             v.Remark,
	}
}

func (m taskTemplateModel) toDomain() domain.TaskTemplate {
	return domain.TaskTemplate{
		ID:                 m.ID,
		Name:               m.Name,
		ProfileID:          m.ProfileID,
		TaskType:           domain.TaskType(m.TaskType),
		PhoneSourceType:    domain.SourceType(m.PhoneSourceType),
		PhoneAPITemplateID: m.PhoneAPITemplateID,
		DefaultTXTDir:      m.DefaultTXTDir,
		CodeSourceType:     domain.SourceType(m.CodeSourceType),
		CodeAPITemplateID:  m.CodeAPITemplateID,
		FailedOutputDir:    m.FailedOutputDir,
		Enabled:            m.Enabled,
		Remark:             m.Remark,
	}
}

type jobModel struct {
	ID                       int64 `gorm:"primaryKey;autoIncrement"`
	ProfileID                int64
	TaskTemplateID           int64
	Name                     string
	TaskType                 string
	PhoneSourceType          string
	CodeSourceType           string
	PhoneSourceConfigJSON    string
	CodeSourceConfigJSON     string
	APITemplateSnapshotJSON  string
	TaskTemplateSnapshotJSON string
	BaseURLSnapshot          string
	ReserveDevicesSnapshot   int64
	IntervalMSSnapshot       int64
	TimeoutMSSnapshot        int64
	CreateDelayMSSnapshot    int64
	Status                   string
	Paused                   bool
	Stopped                  bool
	CreatedAt                time.Time
	StartedAt                *time.Time
	FinishedAt               *time.Time
	UpdatedAt                time.Time
}

func jobModelFromDomain(v domain.Job) jobModel {
	return jobModel{
		ID:                       v.ID,
		ProfileID:                v.ProfileID,
		TaskTemplateID:           v.TaskTemplateID,
		Name:                     v.Name,
		TaskType:                 string(v.TaskType),
		PhoneSourceType:          string(v.PhoneSourceType),
		CodeSourceType:           string(v.CodeSourceType),
		PhoneSourceConfigJSON:    v.PhoneSourceConfigJSON,
		CodeSourceConfigJSON:     v.CodeSourceConfigJSON,
		APITemplateSnapshotJSON:  v.APITemplateSnapshotJSON,
		TaskTemplateSnapshotJSON: v.TaskTemplateSnapshotJSON,
		BaseURLSnapshot:          v.BaseURLSnapshot,
		ReserveDevicesSnapshot:   v.ReserveDevicesSnapshot,
		IntervalMSSnapshot:       v.IntervalSnapshot.Milliseconds(),
		TimeoutMSSnapshot:        v.TimeoutSnapshot.Milliseconds(),
		CreateDelayMSSnapshot:    v.CreateDelaySnapshot.Milliseconds(),
		Status:                   string(v.Status),
		Paused:                   v.Paused,
		Stopped:                  v.Stopped,
		CreatedAt:                v.CreatedAt,
		StartedAt:                v.StartedAt,
		FinishedAt:               v.FinishedAt,
		UpdatedAt:                v.UpdatedAt,
	}
}

func (m jobModel) toDomain() domain.Job {
	return domain.Job{
		ID:                       m.ID,
		ProfileID:                m.ProfileID,
		TaskTemplateID:           m.TaskTemplateID,
		Name:                     m.Name,
		TaskType:                 domain.TaskType(m.TaskType),
		PhoneSourceType:          domain.SourceType(m.PhoneSourceType),
		CodeSourceType:           domain.SourceType(m.CodeSourceType),
		PhoneSourceConfigJSON:    m.PhoneSourceConfigJSON,
		CodeSourceConfigJSON:     m.CodeSourceConfigJSON,
		APITemplateSnapshotJSON:  m.APITemplateSnapshotJSON,
		TaskTemplateSnapshotJSON: m.TaskTemplateSnapshotJSON,
		BaseURLSnapshot:          m.BaseURLSnapshot,
		ReserveDevicesSnapshot:   m.ReserveDevicesSnapshot,
		IntervalSnapshot:         time.Duration(m.IntervalMSSnapshot) * time.Millisecond,
		TimeoutSnapshot:          time.Duration(m.TimeoutMSSnapshot) * time.Millisecond,
		CreateDelaySnapshot:      time.Duration(m.CreateDelayMSSnapshot) * time.Millisecond,
		Status:                   domain.JobStatus(m.Status),
		Paused:                   m.Paused,
		Stopped:                  m.Stopped,
		CreatedAt:                m.CreatedAt,
		StartedAt:                m.StartedAt,
		FinishedAt:               m.FinishedAt,
		UpdatedAt:                m.UpdatedAt,
	}
}

type jobItemModel struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	JobID        int64  `gorm:"index:idx_job_phone,unique"`
	Phone        string `gorm:"index:idx_job_phone,unique"`
	RemoteTaskID uint
	Status       string
	RemoteStatus string
	VerifyCode   string
	LastError    string
	Attempts     int
	SourceLineNo int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func jobItemModelFromDomain(v domain.JobItem) jobItemModel {
	return jobItemModel{
		ID:           v.ID,
		JobID:        v.JobID,
		Phone:        v.Phone,
		RemoteTaskID: v.RemoteTaskID,
		Status:       string(v.Status),
		RemoteStatus: v.RemoteStatus,
		VerifyCode:   v.VerifyCode,
		LastError:    v.LastError,
		Attempts:     v.Attempts,
		SourceLineNo: v.SourceLineNo,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}
}

func (m jobItemModel) toDomain() domain.JobItem {
	return domain.JobItem{
		ID:           m.ID,
		JobID:        m.JobID,
		Phone:        m.Phone,
		RemoteTaskID: m.RemoteTaskID,
		Status:       domain.JobItemStatus(m.Status),
		RemoteStatus: m.RemoteStatus,
		VerifyCode:   m.VerifyCode,
		LastError:    m.LastError,
		Attempts:     m.Attempts,
		SourceLineNo: m.SourceLineNo,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type eventModel struct {
	ID         int64 `gorm:"primaryKey;autoIncrement"`
	JobID      int64
	ItemID     int64
	Phone      string
	Level      string
	EventType  string
	Message    string
	DetailJSON string
	CreatedAt  time.Time
}

type devicePoolSnapshotModel struct {
	ID                  int64 `gorm:"primaryKey;autoIncrement"`
	BaseURL             string
	IdleDeviceCount     int64
	ReserveDevices      int64
	Capacity            int64
	RunningProfileCount int
	RunningJobCount     int
	QueryElapsedMS      int64
	LastError           string
	CreatedAt           time.Time
}

func int64String(v int64) string {
	return strconv.FormatInt(v, 10)
}

func parseInt64(raw string) int64 {
	v, _ := strconv.ParseInt(raw, 10, 64)
	return v
}
