package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	if rec := w.State.activeRecord(); rec != nil {
		return w.syncTask(ctx, rec)
	}

	rec := w.State.nextPendingRecord()
	if rec == nil {
		return nil
	}
	idle, err := w.System.IdleDeviceCount(ctx)
	if err != nil {
		return fmt.Errorf("check idle devices: %w", err)
	}
	capacity := idle - w.IdleThreshold
	w.logger.Printf("idle=%d threshold=%d capacity=%d", idle, w.IdleThreshold, capacity)
	if capacity <= 0 {
		return nil
	}

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
	task, err := w.System.GetTask(ctx, rec.TaskID)
	if err != nil {
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("get task id=%d phone=%s: %w", rec.TaskID, rec.Phone, err)
	}
	return w.handleTask(ctx, rec, task)
}

func (w *Worker) handleTask(ctx context.Context, rec *PhoneRecord, task taskInfo) error {
	if task.Status == "succeeded" || task.FinishedAt != nil && task.Status == "succeeded" {
		rec.Status = recordStatusSucceeded
		rec.LastError = ""
		rec.UpdatedAt = time.Now()
		return w.save()
	}
	if task.Status == "failed" || task.FinishedAt != nil && task.Status != "succeeded" {
		rec.Status = recordStatusFailed
		rec.LastError = task.LastError
		rec.UpdatedAt = time.Now()
		return w.save()
	}
	if rec.Status == recordStatusCodeSubmitted {
		rec.UpdatedAt = time.Now()
		return w.save()
	}
	if !task.NeedPromoterCode {
		rec.Status = recordStatusCreated
		rec.UpdatedAt = time.Now()
		return w.save()
	}

	code, err := w.CodeSource.FetchCode(ctx, rec.Phone)
	if err != nil {
		if errors.Is(err, errCodeNotReady) {
			rec.LastError = "验证码未就绪"
			rec.UpdatedAt = time.Now()
			return w.save()
		}
		rec.LastError = err.Error()
		rec.UpdatedAt = time.Now()
		_ = w.save()
		return fmt.Errorf("fetch code phone=%s: %w", rec.Phone, err)
	}
	if _, err := w.System.SubmitCode(ctx, rec.TaskID, code); err != nil {
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
	return SaveStateFile(w.StatePath, w.State)
}
