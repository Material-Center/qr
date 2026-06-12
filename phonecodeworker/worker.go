package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type workerConfig struct {
	System        *SystemClient
	CodeSource    *CodeSourceClient
	State         *State
	StatePath     string
	IdleThreshold int64
	Interval      time.Duration
	CreateDelay   time.Duration
	Logger        *log.Logger
}

type Worker struct {
	System        *SystemClient
	CodeSource    *CodeSourceClient
	State         *State
	StatePath     string
	IdleThreshold int64
	Interval      time.Duration
	CreateDelay   time.Duration
	logger        *log.Logger
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
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	if cfg.CodeSource != nil {
		cfg.CodeSource.logger = logger
	}
	return &Worker{
		System:        cfg.System,
		CodeSource:    cfg.CodeSource,
		State:         cfg.State,
		StatePath:     cfg.StatePath,
		IdleThreshold: idleThreshold,
		Interval:      interval,
		CreateDelay:   createDelay,
		logger:        logger,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.validate(); err != nil {
		return err
	}
	w.logger.Printf("worker start interval=%s idleThreshold=%d createDelay=%s state=%s records=%d",
		w.Interval, w.IdleThreshold, w.CreateDelay, w.StatePath, len(w.State.Records))
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

func (w *Worker) RunOnce(ctx context.Context) error {
	if err := w.validate(); err != nil {
		return err
	}
	w.logger.Printf("cycle start %s state=%s", w.stateSummary(), w.StatePath)
	if rec := w.State.activeRecord(); rec != nil {
		w.logger.Printf("active task phone=%s task=%d localStatus=%s lastError=%q",
			rec.Phone, rec.TaskID, rec.Status, rec.LastError)
		return w.syncTask(ctx, rec)
	}

	rec := w.State.nextPendingRecord()
	if rec == nil {
		w.logger.Printf("no pending records %s", w.stateSummary())
		return nil
	}
	w.logger.Printf("next pending phone=%s checking idle devices", rec.Phone)
	idle, err := w.System.IdleDeviceCount(ctx)
	if err != nil {
		return fmt.Errorf("check idle devices: %w", err)
	}
	capacity := idle - w.IdleThreshold
	w.logger.Printf("idle=%d threshold=%d capacity=%d", idle, w.IdleThreshold, capacity)
	if capacity <= 0 {
		w.logger.Printf("waiting idle capacity phone=%s idle=%d threshold=%d nextCheckIn=%s",
			rec.Phone, idle, w.IdleThreshold, w.Interval)
		return nil
	}

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
	rec.UpdatedAt = time.Now()
	if err := w.save(); err != nil {
		return err
	}
	w.logger.Printf("created receive task id=%d phone=%s", task.ID, rec.Phone)
	return w.handleTask(ctx, rec, task)
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
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("submit code task=%d phone=%s: %w", rec.TaskID, rec.Phone, err)
	}
	rec.Status = recordStatusCodeSubmitted
	rec.VerifyCode = code
	rec.LastError = ""
	rec.UpdatedAt = time.Now()
	w.logger.Printf("submitted code task=%d phone=%s", rec.TaskID, rec.Phone)
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
	w.logger.Printf("state saved path=%s %s", w.StatePath, w.stateSummary())
	return nil
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
