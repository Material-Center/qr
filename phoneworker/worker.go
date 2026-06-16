package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type workerConfig struct {
	System        *SystemClient
	PhoneSource   *PhoneSourceClient
	PauseFile     string
	IdleThreshold int64
	Interval      time.Duration
	CreateDelay   time.Duration
	Logger        *log.Logger
}

type Worker struct {
	System        *SystemClient
	PhoneSource   *PhoneSourceClient
	PauseFile     string
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
	createDelay := cfg.CreateDelay
	if createDelay < 0 {
		createDelay = 0
	}
	idleThreshold := cfg.IdleThreshold
	if idleThreshold < 0 {
		idleThreshold = 0
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	if cfg.System != nil {
		cfg.System.logger = logger
	}
	if cfg.PhoneSource != nil {
		cfg.PhoneSource.logger = logger
	}
	return &Worker{
		System:        cfg.System,
		PhoneSource:   cfg.PhoneSource,
		PauseFile:     strings.TrimSpace(cfg.PauseFile),
		IdleThreshold: idleThreshold,
		Interval:      interval,
		CreateDelay:   createDelay,
		logger:        logger,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if w.System == nil || w.PhoneSource == nil {
		return fmt.Errorf("worker clients are required")
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
	if w.isPaused() {
		w.logger.Printf("paused pauseFile=%s", w.PauseFile)
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

	phone, err := w.PhoneSource.FetchPhone(ctx)
	if err != nil {
		return err
	}
	w.logger.Printf("create task start phone=%s mode=%s createDelay=%s", phone, smsReceiveModeUserSent, w.CreateDelay)
	taskID, err := w.System.CreateUserSentTask(ctx, phone, w.CreateDelay)
	if err != nil {
		w.logger.Printf("create task error phone=%s mode=%s createDelay=%s err=%v", phone, smsReceiveModeUserSent, w.CreateDelay, err)
		return fmt.Errorf("create task phone=%s: %w", phone, err)
	}
	w.logger.Printf("created task id=%d phone=%s mode=%s", taskID, phone, smsReceiveModeUserSent)
	return nil
}

func (w *Worker) WaitIdle(ctx context.Context) error {
	return nil
}

func (w *Worker) isPaused() bool {
	if strings.TrimSpace(w.PauseFile) == "" {
		return false
	}
	_, err := os.Stat(w.PauseFile)
	return err == nil
}
