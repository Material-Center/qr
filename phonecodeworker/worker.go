package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultMaxTaskRetries = 1

	phoneRegisterStatusCodeVerifyCodeTimeout = 1002
	phoneRegisterStatusCodeHeartbeatTimeout  = 1003
	phoneRegisterStatusCodeTaskTimeout       = 1004
)

type workerConfig struct {
	System         *SystemClient
	CodeSource     *CodeSourceClient
	State          *State
	StatePath      string
	FailedPath     string
	PauseFile      string
	IdleThreshold  int64
	Interval       time.Duration
	CreateDelay    time.Duration
	MaxTaskRetries int
	Logger         *log.Logger
}

type Worker struct {
	System         *SystemClient
	CodeSource     *CodeSourceClient
	State          *State
	StatePath      string
	FailedPath     string
	PauseFile      string
	IdleThreshold  int64
	Interval       time.Duration
	CreateDelay    time.Duration
	MaxTaskRetries int
	logger         *log.Logger
}

func NewWorker(cfg workerConfig) *Worker {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	idleThreshold := cfg.IdleThreshold
	if idleThreshold < 0 {
		idleThreshold = 0
	}
	createDelay := cfg.CreateDelay
	if createDelay < 0 {
		createDelay = 0
	}
	maxTaskRetries := cfg.MaxTaskRetries
	if maxTaskRetries == 0 {
		maxTaskRetries = defaultMaxTaskRetries
	}
	if maxTaskRetries < 0 {
		maxTaskRetries = 0
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	if cfg.CodeSource != nil {
		cfg.CodeSource.logger = logger
	}
	return &Worker{
		System:         cfg.System,
		CodeSource:     cfg.CodeSource,
		State:          cfg.State,
		StatePath:      cfg.StatePath,
		FailedPath:     strings.TrimSpace(cfg.FailedPath),
		PauseFile:      strings.TrimSpace(cfg.PauseFile),
		IdleThreshold:  idleThreshold,
		Interval:       interval,
		CreateDelay:    createDelay,
		MaxTaskRetries: maxTaskRetries,
		logger:         logger,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.validate(); err != nil {
		return err
	}
	w.logger.Printf("worker start interval=%s idleThreshold=%d createDelay=%s maxTaskRetries=%d state=%s records=%d",
		w.Interval, w.IdleThreshold, w.CreateDelay, w.MaxTaskRetries, w.StatePath, len(w.State.Records))
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			w.logger.Printf("cycle error: %v", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) (err error) {
	if err := w.validate(); err != nil {
		return err
	}
	defer func() {
		if exportErr := w.exportFailedImportFile(); exportErr != nil {
			w.logger.Printf("failed import export error path=%s err=%v", w.FailedPath, exportErr)
			if err == nil {
				err = exportErr
			}
		}
	}()
	w.logger.Printf("cycle start %s state=%s", w.stateSummary(), w.StatePath)
	var firstErr error
	active := w.State.activeRecords()
	for _, rec := range active {
		w.logger.Printf("active task phone=%s task=%d localStatus=%s lastError=%q",
			rec.Phone, rec.TaskID, rec.Status, rec.LastError)
		if err := w.syncTask(ctx, rec); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			w.logger.Printf("active task sync error phone=%s task=%d err=%v", rec.Phone, rec.TaskID, err)
		}
	}

	pendingCount := len(w.State.pendingRecords(0))
	if pendingCount == 0 {
		w.logger.Printf("no pending records %s", w.stateSummary())
		return firstErr
	}
	if w.isPaused() {
		w.logger.Printf("paused pauseFile=%s pending=%d active=%d",
			w.PauseFile, pendingCount, len(w.State.activeRecords()))
		return firstErr
	}
	w.logger.Printf("pending records=%d checking idle devices", pendingCount)
	idle, err := w.System.IdleDeviceCount(ctx)
	if err != nil {
		if firstErr != nil {
			w.logger.Printf("check idle devices error after sync error: %v", err)
			return firstErr
		}
		return fmt.Errorf("check idle devices: %w", err)
	}
	capacity := idle - w.IdleThreshold
	activeCount := len(w.State.activeRecords())
	slots := capacity
	w.logger.Printf("idle=%d threshold=%d capacity=%d active=%d slots=%d pending=%d",
		idle, w.IdleThreshold, capacity, activeCount, slots, pendingCount)
	if capacity <= 0 {
		w.logger.Printf("waiting idle capacity idle=%d threshold=%d nextCheckIn=%s",
			idle, w.IdleThreshold, w.Interval)
		return firstErr
	}

	for i, rec := range w.State.pendingRecords(int(slots)) {
		if i > 0 {
			w.logger.Printf("waiting create interval=%s before next pending phone=%s", w.Interval, rec.Phone)
			if err := waitDuration(ctx, w.Interval); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				break
			}
			idle, err := w.System.IdleDeviceCount(ctx)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("check idle devices before next create: %w", err)
				}
				w.logger.Printf("check idle devices before next create error phone=%s err=%v", rec.Phone, err)
				break
			}
			capacity := idle - w.IdleThreshold
			activeCount := len(w.State.activeRecords())
			w.logger.Printf("post-wait idle=%d threshold=%d capacity=%d active=%d nextPhone=%s",
				idle, w.IdleThreshold, capacity, activeCount, rec.Phone)
			if capacity <= 0 {
				w.logger.Printf("stop creating pending records no idle capacity phone=%s idle=%d threshold=%d",
					rec.Phone, idle, w.IdleThreshold)
				break
			}
		}
		if err := w.createAndHandleTask(ctx, rec); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			w.logger.Printf("create task error phone=%s err=%v", rec.Phone, err)
		}
	}
	if err := w.retryCodeNotReadyRecords(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (w *Worker) createAndHandleTask(ctx context.Context, rec *PhoneRecord) error {
	w.logger.Printf("creating receive task phone=%s createDelay=%s", rec.Phone, w.CreateDelay)
	task, err := w.System.CreateReceiveTask(ctx, rec.Phone, w.CreateDelay)
	if err != nil {
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("create receive task phone=%s: %w", rec.Phone, err)
	}
	rec.TaskID = task.ID
	rec.Status = recordStatusCreated
	rec.LastError = ""
	rec.VerifyCode = ""
	rec.TaskAttempts++
	rec.UpdatedAt = time.Now()
	if err := w.save(); err != nil {
		return err
	}
	w.logger.Printf("created receive task id=%d phone=%s", task.ID, rec.Phone)
	return w.handleTask(ctx, rec, task)
}

func (w *Worker) retryCodeNotReadyRecords(ctx context.Context) error {
	var firstErr error
	for _, rec := range w.State.activeRecords() {
		if rec.LastError != "验证码未就绪" {
			continue
		}
		w.logger.Printf("retry code not ready task=%d phone=%s after create batch", rec.TaskID, rec.Phone)
		if err := w.syncTask(ctx, rec); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			w.logger.Printf("retry code not ready error task=%d phone=%s err=%v", rec.TaskID, rec.Phone, err)
		}
	}
	return firstErr
}

func waitDuration(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (w *Worker) syncTask(ctx context.Context, rec *PhoneRecord) error {
	w.logger.Printf("sync task phone=%s task=%d localStatus=%s", rec.Phone, rec.TaskID, rec.Status)
	task, err := w.System.GetTask(ctx, rec.TaskID)
	if err != nil {
		if isMissingTaskError(err) {
			w.logger.Printf("active task missing task=%d phone=%s err=%v", rec.TaskID, rec.Phone, err)
			rec.Status = recordStatusFailed
			rec.LastError = err.Error()
			rec.UpdatedAt = time.Now()
			return w.save()
		}
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("get task id=%d phone=%s: %w", rec.TaskID, rec.Phone, err)
	}
	return w.handleTask(ctx, rec, task)
}

func (w *Worker) handleTask(ctx context.Context, rec *PhoneRecord, task taskInfo) error {
	w.logger.Printf("task status task=%d phone=%s remoteStatus=%s needPromoterCode=%t statusCode=%s finished=%t lastError=%q",
		task.ID, rec.Phone, task.Status, task.NeedPromoterCode, formatStatusCode(task.StatusCode), task.FinishedAt != nil, task.LastError)
	if task.Status == "succeeded" || task.FinishedAt != nil && task.Status == "succeeded" {
		w.logger.Printf("task succeeded task=%d phone=%s", rec.TaskID, rec.Phone)
		rec.Status = recordStatusSucceeded
		rec.LastError = ""
		rec.UpdatedAt = time.Now()
		return w.save()
	}
	if task.Status == "failed" || task.FinishedAt != nil && task.Status != "succeeded" {
		w.logger.Printf("task failed task=%d phone=%s lastError=%q", rec.TaskID, rec.Phone, task.LastError)
		if w.isRetryableTimeoutTask(rec, task) {
			return w.resetTaskForRetry(rec, task.LastError)
		}
		rec.Status = recordStatusFailed
		rec.LastError = task.LastError
		rec.UpdatedAt = time.Now()
		return w.save()
	}
	if rec.Status == recordStatusCodeSubmitted {
		w.logger.Printf("code already submitted task=%d phone=%s waiting remote result", rec.TaskID, rec.Phone)
		rec.UpdatedAt = time.Now()
		return w.save()
	}
	if !task.NeedPromoterCode {
		w.logger.Printf("waiting promoter code task=%d phone=%s remoteStatus=%s lastError=%q",
			rec.TaskID, rec.Phone, task.Status, task.LastError)
		rec.Status = recordStatusCreated
		rec.UpdatedAt = time.Now()
		return w.save()
	}

	w.logger.Printf("promoter code needed task=%d phone=%s fetching code", rec.TaskID, rec.Phone)
	code, err := w.CodeSource.FetchCode(ctx, rec.Phone)
	if err != nil {
		if errors.Is(err, errCodeNotReady) {
			w.logger.Printf("code not ready task=%d phone=%s nextCheckIn=%s", rec.TaskID, rec.Phone, w.Interval)
			rec.LastError = "验证码未就绪"
			rec.UpdatedAt = time.Now()
			return w.save()
		}
		w.logger.Printf("fetch code error task=%d phone=%s err=%v", rec.TaskID, rec.Phone, err)
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("fetch code phone=%s: %w", rec.Phone, err)
	}
	w.logger.Printf("submitting code task=%d phone=%s code=%s", rec.TaskID, rec.Phone, code)
	if _, err := w.System.SubmitCode(ctx, rec.TaskID, code); err != nil {
		w.logger.Printf("submit code error task=%d phone=%s err=%v", rec.TaskID, rec.Phone, err)
		if w.isRetryableTimeoutError(rec, err) {
			return w.resetTaskForRetry(rec, err.Error())
		}
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("submit code task=%d phone=%s: %w", rec.TaskID, rec.Phone, err)
	}
	rec.Status = recordStatusSucceeded
	rec.VerifyCode = code
	rec.LastError = ""
	rec.UpdatedAt = time.Now()
	w.logger.Printf("submitted code task=%d phone=%s mark succeeded", rec.TaskID, rec.Phone)
	return w.save()
}

func (w *Worker) isRetryableTimeoutTask(rec *PhoneRecord, task taskInfo) bool {
	if rec == nil || !isTaskTimeoutFailure(task) {
		return false
	}
	return rec.timeoutRetryCount() < w.MaxTaskRetries
}

func (w *Worker) isRetryableTimeoutError(rec *PhoneRecord, err error) bool {
	if rec == nil || !isTaskTimeoutError(err) {
		return false
	}
	return rec.timeoutRetryCount() < w.MaxTaskRetries
}

func (w *Worker) resetTaskForRetry(rec *PhoneRecord, reason string) error {
	oldTaskID := rec.TaskID
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "服务端任务超时"
	}
	rec.TaskID = 0
	rec.Status = recordStatusPending
	rec.VerifyCode = ""
	rec.LastError = reason
	rec.UpdatedAt = time.Now()
	w.logger.Printf("server task timed out phone=%s oldTask=%d retry=%d/%d reason=%q",
		rec.Phone, oldTaskID, rec.timeoutRetryCount()+1, w.MaxTaskRetries, rec.LastError)
	return w.save()
}

func (w *Worker) validate() error {
	if w.System == nil || w.CodeSource == nil || w.State == nil {
		return fmt.Errorf("worker clients and state are required")
	}
	if w.StatePath == "" {
		return fmt.Errorf("state path is required")
	}
	return nil
}

func (w *Worker) save() error {
	if err := SaveStateFile(w.StatePath, w.State); err != nil {
		w.logger.Printf("state save failed path=%s err=%v", w.StatePath, err)
		return err
	}
	if err := w.exportFailedImportFile(); err != nil {
		return err
	}
	w.logger.Printf("state saved path=%s %s", w.StatePath, w.stateSummary())
	return nil
}

func (w *Worker) exportFailedImportFile() error {
	if strings.TrimSpace(w.FailedPath) == "" {
		return nil
	}
	if err := SaveFailedImportFile(w.FailedPath, w.State); err != nil {
		return err
	}
	failedCount := len(w.State.failedPhones())
	if failedCount > 0 {
		w.logger.Printf("failed import file updated path=%s failed=%d", w.FailedPath, failedCount)
	}
	return nil
}

func (w *Worker) isPaused() bool {
	if strings.TrimSpace(w.PauseFile) == "" {
		return false
	}
	_, err := os.Stat(w.PauseFile)
	return err == nil
}

func (w *Worker) stateSummary() string {
	counts := map[string]int{}
	for _, rec := range w.State.Records {
		status := strings.TrimSpace(rec.Status)
		if status == "" {
			status = recordStatusPending
		}
		counts[status]++
	}
	return fmt.Sprintf("records=%d pending=%d created=%d codeSubmitted=%d succeeded=%d failed=%d",
		len(w.State.Records),
		counts[recordStatusPending],
		counts[recordStatusCreated],
		counts[recordStatusCodeSubmitted],
		counts[recordStatusSucceeded],
		counts[recordStatusFailed],
	)
}

func formatStatusCode(statusCode *int) string {
	if statusCode == nil {
		return "-"
	}
	return fmt.Sprint(*statusCode)
}

func isMissingTaskError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "任务不存在")
}

func (rec *PhoneRecord) timeoutRetryCount() int {
	if rec == nil || rec.TaskAttempts <= 0 {
		return 0
	}
	return rec.TaskAttempts - 1
}

func isTaskTimeoutFailure(task taskInfo) bool {
	if task.StatusCode != nil {
		switch *task.StatusCode {
		case phoneRegisterStatusCodeVerifyCodeTimeout,
			phoneRegisterStatusCodeHeartbeatTimeout,
			phoneRegisterStatusCodeTaskTimeout:
			return true
		}
	}
	return isTimeoutText(task.LastError)
}

func isTaskTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "任务已超时") || strings.Contains(err.Error(), "验证码已超时")
}

func isTimeoutText(text string) bool {
	text = strings.TrimSpace(text)
	return strings.Contains(text, "任务总超时") ||
		strings.Contains(text, "验证码等待超时") ||
		strings.Contains(text, "设备心跳超时")
}
