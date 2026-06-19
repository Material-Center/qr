package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"phone-task-client/internal/backend"
	"phone-task-client/internal/core"
	"phone-task-client/internal/domain"
	phoneexport "phone-task-client/internal/export"
	"phone-task-client/internal/source"
	"phone-task-client/internal/store"
)

const defaultSystemBaseURL = "http://210.16.170.132:1111/api"
const defaultPhoneSourceURL = "http://206.238.179.123:37520/OPenApi/GetOrder?infor=vwZt5p4FmyeupCqMqKsC2ktcpczoBuX23akOGMEPlsw%3D&project=wb3"
const defaultCodeSourceURL = "https://q8.qq0.lol/api/imla?t=u2lX6gNl&phone={phone}"

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

type App struct {
	ctx    context.Context
	store  *store.Store
	dbPath string
	mu     sync.Mutex
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

type AppStatus struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	GitCommit   string `json:"gitCommit"`
	BuildTime   string `json:"buildTime"`
	Runtime     string `json:"runtime"`
	CoreReady   bool   `json:"coreReady"`
	Storage     string `json:"storage"`
	DBPath      string `json:"dbPath"`
	Description string `json:"description"`
}

type Dashboard struct {
	Status        AppStatus             `json:"status"`
	Settings      domain.GlobalSettings `json:"settings"`
	Profiles      []domain.Profile      `json:"profiles"`
	APITemplates  []domain.APITemplate  `json:"apiTemplates"`
	TaskTemplates []domain.TaskTemplate `json:"taskTemplates"`
	Jobs          []JobSummary          `json:"jobs"`
}

type JobSummary struct {
	Job       domain.Job `json:"job"`
	Total     int        `json:"total"`
	Pending   int        `json:"pending"`
	Active    int        `json:"active"`
	Succeeded int        `json:"succeeded"`
	Failed    int        `json:"failed"`
}

type StartJobRequest struct {
	Name               string `json:"name"`
	ProfileID          int64  `json:"profileId"`
	TaskTemplateID     int64  `json:"taskTemplateId"`
	TaskType           string `json:"taskType"`
	PhoneSourceType    string `json:"phoneSourceType"`
	InputPath          string `json:"inputPath"`
	PhoneAPITemplateID int64  `json:"phoneApiTemplateId"`
	CodeAPITemplateID  int64  `json:"codeApiTemplateId"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	path := defaultDBPath()
	st, err := store.Open(path)
	if err != nil {
		println("open store:", err.Error())
		return
	}
	a.store = st
	a.dbPath = path
	settings, _ := st.LoadGlobalSettings()
	changed := false
	if strings.TrimSpace(settings.BaseURL) == "" {
		settings.BaseURL = defaultSystemBaseURL
		changed = true
	}
	if settings.Interval <= 0 {
		settings.Interval = 3 * time.Second
		changed = true
	}
	if settings.Timeout <= 0 {
		settings.Timeout = 10 * time.Second
		changed = true
	}
	if strings.TrimSpace(settings.LogDir) == "" {
		settings.LogDir = filepath.Join(filepath.Dir(path), "logs")
		changed = true
	}
	if changed {
		_ = st.SaveGlobalSettings(settings)
	}
	if err := ensureDefaultAPITemplates(st); err != nil {
		println("seed api templates:", err.Error())
	}
	if jobs, err := st.ListRunnableJobs(); err == nil && len(jobs) > 0 {
		a.startLoop()
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	cancel := a.cancel
	a.cancel = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	a.wg.Wait()
	if a.store != nil {
		_ = a.store.Close()
	}
}

func (a *App) Status() AppStatus {
	return AppStatus{
		Name:        "Phone Task Client",
		Version:     version,
		GitCommit:   gitCommit,
		BuildTime:   buildTime,
		Runtime:     runtime.GOOS + "/" + runtime.GOARCH,
		CoreReady:   a.store != nil,
		Storage:     "SQLite",
		DBPath:      a.dbPath,
		Description: "支持发码、收码、TXT/API 来源、共享设备池、状态恢复、暂停/继续/停止和失败导出。",
	}
}

func (a *App) Dashboard() (Dashboard, error) {
	if err := a.ensureStore(); err != nil {
		return Dashboard{}, err
	}
	settings, err := a.store.LoadGlobalSettings()
	if err != nil {
		return Dashboard{}, err
	}
	profiles, err := a.store.ListProfiles()
	if err != nil {
		return Dashboard{}, err
	}
	apiTemplates, err := a.store.ListAPITemplates()
	if err != nil {
		return Dashboard{}, err
	}
	taskTemplates, err := a.store.ListTaskTemplates()
	if err != nil {
		return Dashboard{}, err
	}
	jobs, err := a.jobSummaries(50)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		Status:        a.Status(),
		Settings:      settings,
		Profiles:      profiles,
		APITemplates:  apiTemplates,
		TaskTemplates: taskTemplates,
		Jobs:          jobs,
	}, nil
}

func (a *App) SaveSettings(settings domain.GlobalSettings) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	if settings.Interval <= 0 {
		settings.Interval = 3 * time.Second
	}
	if settings.Timeout <= 0 {
		settings.Timeout = 10 * time.Second
	}
	return a.store.SaveGlobalSettings(settings)
}

func (a *App) SaveProfile(profile domain.Profile) (domain.Profile, error) {
	if err := a.ensureStore(); err != nil {
		return domain.Profile{}, err
	}
	profile.Enabled = true
	if profile.TokenMask == "" {
		profile.TokenMask = maskToken(profile.TokenRef)
	}
	return a.store.SaveProfile(profile)
}

func (a *App) SaveAPITemplate(t domain.APITemplate) (domain.APITemplate, error) {
	if err := a.ensureStore(); err != nil {
		return domain.APITemplate{}, err
	}
	if t.Method == "" {
		t.Method = domain.HTTPMethodGET
	}
	if t.ResponseType == "" {
		t.ResponseType = domain.ResponseTypeAuto
	}
	t.Enabled = true
	return a.store.SaveAPITemplate(t)
}

func (a *App) SaveTaskTemplate(t domain.TaskTemplate) (domain.TaskTemplate, error) {
	if err := a.ensureStore(); err != nil {
		return domain.TaskTemplate{}, err
	}
	t.Enabled = true
	return a.store.SaveTaskTemplate(t)
}

func (a *App) StartJob(req StartJobRequest) (domain.Job, error) {
	if err := a.ensureStore(); err != nil {
		return domain.Job{}, err
	}
	job, err := a.createJob(req)
	if err != nil {
		return domain.Job{}, err
	}
	a.startLoop()
	return job, nil
}

func (a *App) RunJob(jobID int64) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	job, err := a.store.GetJob(jobID)
	if err != nil {
		return err
	}
	job.Paused = false
	job.Stopped = false
	job.Status = domain.JobStatusRunning
	job.UpdatedAt = time.Now()
	if err := a.store.UpdateJob(job); err != nil {
		return err
	}
	a.startLoop()
	return nil
}

func (a *App) PauseJob(jobID int64) error {
	return a.setJobControl(jobID, "pause")
}

func (a *App) ResumeJob(jobID int64) error {
	if err := a.setJobControl(jobID, "resume"); err != nil {
		return err
	}
	a.startLoop()
	return nil
}

func (a *App) StopJob(jobID int64) error {
	return a.setJobControl(jobID, "stop")
}

func (a *App) ListJobItems(jobID int64) ([]domain.JobItem, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListJobItems(jobID)
}

func (a *App) ExportFailed(jobID int64, path string) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	job, err := a.store.GetJob(jobID)
	if err != nil {
		return err
	}
	items, err := a.store.ListJobItems(jobID)
	if err != nil {
		return err
	}
	raw, err := phoneexport.BuildFailedRetryFile(job, items)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

func (a *App) createJob(req StartJobRequest) (domain.Job, error) {
	profile, err := a.store.GetProfile(req.ProfileID)
	if err != nil {
		return domain.Job{}, err
	}
	settings, err := a.store.LoadGlobalSettings()
	if err != nil {
		return domain.Job{}, err
	}
	taskType := domain.TaskType(req.TaskType)
	phoneSourceType := domain.SourceType(req.PhoneSourceType)
	var phoneTemplate domain.APITemplate
	var codeTemplate domain.APITemplate
	codeSourceType := domain.SourceTypeNone
	if req.TaskTemplateID > 0 {
		tmpl, err := a.store.GetTaskTemplate(req.TaskTemplateID)
		if err != nil {
			return domain.Job{}, err
		}
		if taskType == "" {
			taskType = tmpl.TaskType
		}
		if phoneSourceType == "" {
			phoneSourceType = tmpl.PhoneSourceType
		}
		if req.PhoneAPITemplateID == 0 {
			req.PhoneAPITemplateID = tmpl.PhoneAPITemplateID
		}
		if req.CodeAPITemplateID == 0 {
			req.CodeAPITemplateID = tmpl.CodeAPITemplateID
		}
	}
	if taskType == "" {
		taskType = domain.TaskTypeReceiveCode
	}
	if taskType != domain.TaskTypeReceiveCode && taskType != domain.TaskTypeSendCode {
		return domain.Job{}, fmt.Errorf("任务模式不支持: %s", taskType)
	}
	if phoneSourceType == "" {
		phoneSourceType = domain.SourceTypeTXT
	}
	if phoneSourceType != domain.SourceTypeTXT && phoneSourceType != domain.SourceTypeAPI {
		return domain.Job{}, fmt.Errorf("手机号来源不支持: %s", phoneSourceType)
	}
	if phoneSourceType == domain.SourceTypeAPI {
		if req.PhoneAPITemplateID == 0 {
			return domain.Job{}, fmt.Errorf("手机号 API 模板不能为空")
		}
		phoneTemplate, err = a.store.GetAPITemplate(req.PhoneAPITemplateID)
		if err != nil {
			return domain.Job{}, err
		}
	}
	if taskType == domain.TaskTypeReceiveCode {
		if req.CodeAPITemplateID == 0 {
			return domain.Job{}, fmt.Errorf("验证码 API 模板不能为空")
		}
		codeSourceType = domain.SourceTypeAPI
		codeTemplate, err = a.store.GetAPITemplate(req.CodeAPITemplateID)
		if err != nil {
			return domain.Job{}, err
		}
	}
	now := time.Now()
	var items []domain.JobItem
	if phoneSourceType == domain.SourceTypeTXT {
		if strings.TrimSpace(req.InputPath) == "" {
			return domain.Job{}, fmt.Errorf("TXT 文件路径不能为空")
		}
		raw, err := os.ReadFile(req.InputPath)
		if err != nil {
			return domain.Job{}, err
		}
		parsed, err := source.ParseTXTImport(string(raw), false)
		if err != nil {
			return domain.Job{}, err
		}
		for _, entry := range parsed.Phones {
			items = append(items, domain.JobItem{
				Phone:        entry.Phone,
				Status:       domain.JobItemStatusPending,
				SourceLineNo: entry.LineNo,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
	}
	phoneConfig, _ := json.Marshal(phoneTemplate)
	codeConfig, _ := json.Marshal(codeTemplate)
	baseURL := settings.BaseURL
	if strings.TrimSpace(profile.BaseURLOverride) != "" {
		baseURL = profile.BaseURLOverride
	}
	if strings.TrimSpace(baseURL) == "" {
		return domain.Job{}, fmt.Errorf("服务器地址不能为空")
	}
	if strings.TrimSpace(profile.TokenRef) == "" {
		return domain.Job{}, fmt.Errorf("用户 Token 不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = string(taskType) + "-" + now.Format("20060102-150405")
	}
	job := domain.Job{
		ProfileID:              profile.ID,
		TaskTemplateID:         req.TaskTemplateID,
		Name:                   name,
		TaskType:               taskType,
		PhoneSourceType:        phoneSourceType,
		CodeSourceType:         codeSourceType,
		PhoneSourceConfigJSON:  string(phoneConfig),
		CodeSourceConfigJSON:   string(codeConfig),
		BaseURLSnapshot:        strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		ReserveDevicesSnapshot: settings.ReserveDevices,
		IntervalSnapshot:       settings.Interval,
		TimeoutSnapshot:        settings.Timeout,
		CreateDelaySnapshot:    profile.CreateDelay,
		Status:                 domain.JobStatusRunning,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	job, _, err = a.store.CreateJob(job, items)
	return job, err
}

func (a *App) startLoop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer func() {
			a.mu.Lock()
			a.cancel = nil
			a.mu.Unlock()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := a.runOnce(ctx); err != nil {
				println("runner error:", err.Error())
			}
			timer := time.NewTimer(a.loopInterval())
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
}

func (a *App) runOnce(ctx context.Context) error {
	jobs, err := a.store.ListRunnableJobs()
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}
	settings, _ := a.store.LoadGlobalSettings()
	sharedProfile, _ := a.store.GetProfile(jobs[0].ProfileID)
	shared := backend.NewSystemClient(jobs[0].BaseURLSnapshot, sharedProfile.TokenRef, settings.Timeout)
	apiSource := source.NewAPIClient(settings.Timeout)
	runner := core.NewRunner(a.store, shared, apiSource, time.Now).WithBackendForJob(func(job domain.Job) core.Backend {
		profile, err := a.store.GetProfile(job.ProfileID)
		if err != nil {
			return shared
		}
		return backend.NewSystemClient(job.BaseURLSnapshot, profile.TokenRef, job.TimeoutSnapshot)
	})
	return runner.RunOnce(ctx)
}

func (a *App) loopInterval() time.Duration {
	settings, err := a.store.LoadGlobalSettings()
	if err != nil || settings.Interval <= 0 {
		return 3 * time.Second
	}
	return settings.Interval
}

func (a *App) setJobControl(jobID int64, action string) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	job, err := a.store.GetJob(jobID)
	if err != nil {
		return err
	}
	switch action {
	case "pause":
		job.Paused = true
		job.Status = domain.JobStatusPaused
	case "resume":
		job.Paused = false
		job.Stopped = false
		job.Status = domain.JobStatusRunning
	case "stop":
		job.Stopped = true
		job.Status = domain.JobStatusStopped
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
	job.UpdatedAt = time.Now()
	return a.store.UpdateJob(job)
}

func (a *App) jobSummaries(limit int) ([]JobSummary, error) {
	jobs, err := a.store.ListJobs(limit)
	if err != nil {
		return nil, err
	}
	out := make([]JobSummary, 0, len(jobs))
	for _, job := range jobs {
		items, err := a.store.ListJobItems(job.ID)
		if err != nil {
			return nil, err
		}
		summary := JobSummary{Job: job, Total: len(items)}
		for _, item := range items {
			switch item.Status {
			case domain.JobItemStatusPending:
				summary.Pending++
			case domain.JobItemStatusCreated, domain.JobItemStatusWaitingCode, domain.JobItemStatusCodeSubmitted:
				summary.Active++
			case domain.JobItemStatusSucceeded:
				summary.Succeeded++
			case domain.JobItemStatusFailed:
				summary.Failed++
			}
		}
		out = append(out, summary)
	}
	return out, nil
}

func (a *App) ensureStore() error {
	if a.store == nil {
		return fmt.Errorf("store not ready")
	}
	return nil
}

func defaultDBPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	path := filepath.Join(dir, "phone-task-client")
	_ = os.MkdirAll(path, 0o755)
	return filepath.Join(path, "client.db")
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func ensureDefaultAPITemplates(st *store.Store) error {
	templates, err := st.ListAPITemplates()
	if err != nil {
		return err
	}
	defaults := []domain.APITemplate{
		{
			Name:         "默认发码取号 API",
			APIType:      domain.APITypePhoneSource,
			Method:       domain.HTTPMethodGET,
			URL:          defaultPhoneSourceURL,
			ResponseType: domain.ResponseTypeAuto,
			Enabled:      true,
			Remark:       "兼容 phoneworker 默认取号接口。",
		},
		{
			Name:         "默认收码验证码 API",
			APIType:      domain.APITypeCodeSource,
			Method:       domain.HTTPMethodGET,
			URL:          defaultCodeSourceURL,
			ResponseType: domain.ResponseTypeAuto,
			Enabled:      true,
			Remark:       "兼容 phonecodeworker 导入文件第一行验证码接口。",
		},
	}
	for _, item := range defaults {
		if hasAPITemplate(templates, item.APIType, item.URL) {
			continue
		}
		if _, err := st.SaveAPITemplate(item); err != nil {
			return err
		}
	}
	return nil
}

func hasAPITemplate(items []domain.APITemplate, apiType domain.APIType, url string) bool {
	url = strings.TrimSpace(url)
	for _, item := range items {
		if item.APIType == apiType && strings.TrimSpace(item.URL) == url {
			return true
		}
	}
	return false
}
