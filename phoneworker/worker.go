package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type workerConfig struct {
	System        *SystemClient
	PhoneSource   *PhoneSourceClient
	IdleThreshold int64
	Interval      time.Duration
	Logger        *log.Logger
}

type Worker struct {
	System        *SystemClient
	PhoneSource   *PhoneSourceClient
	IdleThreshold int64
	Interval      time.Duration
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
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Worker{
		System:        cfg.System,
		PhoneSource:   cfg.PhoneSource,
		IdleThreshold: idleThreshold,
		Interval:      interval,
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
	idle, err := w.System.IdleDeviceCount(ctx)
	if err != nil {
		return fmt.Errorf("check idle devices: %w", err)
	}
	w.logger.Printf("idle=%d threshold=%d", idle, w.IdleThreshold)
	if idle <= w.IdleThreshold {
		return nil
	}

	phone, err := w.PhoneSource.FetchPhone(ctx)
	if err != nil {
		return err
	}
	taskID, err := w.System.CreateUserSentTask(ctx, phone)
	if err != nil {
		return fmt.Errorf("create task phone=%s: %w", phone, err)
	}
	w.logger.Printf("created task id=%d phone=%s mode=%s", taskID, phone, smsReceiveModeUserSent)
	return nil
}
