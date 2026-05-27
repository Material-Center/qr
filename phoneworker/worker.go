package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type workerConfig struct {
	System        *SystemClient
	PhoneSource   *PhoneSourceClient
	IdleThreshold int64
	Interval      time.Duration
	CreateDelay   time.Duration
	Logger        *log.Logger
}

type Worker struct {
	System        *SystemClient
	PhoneSource   *PhoneSourceClient
	IdleThreshold int64
	Interval      time.Duration
	CreateDelay   time.Duration
	logger        *log.Logger
	inFlight      atomic.Int64
	createMu      sync.Mutex
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
	return &Worker{
		System:        cfg.System,
		PhoneSource:   cfg.PhoneSource,
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
	idle, err := w.System.IdleDeviceCount(ctx)
	if err != nil {
		return fmt.Errorf("check idle devices: %w", err)
	}
	inFlight := w.inFlight.Load()
	capacity := idle - w.IdleThreshold - inFlight
	w.logger.Printf("idle=%d threshold=%d inFlight=%d capacity=%d", idle, w.IdleThreshold, inFlight, capacity)
	if capacity <= 0 {
		return nil
	}

	phone, err := w.PhoneSource.FetchPhone(ctx)
	if err != nil {
		return err
	}
	if w.CreateDelay > 0 {
		w.scheduleDelayedCreate(ctx, phone)
		return nil
	}
	taskID, err := w.System.CreateUserSentTask(ctx, phone)
	if err != nil {
		return fmt.Errorf("create task phone=%s: %w", phone, err)
	}
	w.logger.Printf("created task id=%d phone=%s mode=%s", taskID, phone, smsReceiveModeUserSent)
	return nil
}

func (w *Worker) scheduleDelayedCreate(ctx context.Context, phone string) {
	w.inFlight.Add(1)
	w.logger.Printf("phone=%s scheduled create delay=%s", phone, w.CreateDelay)
	go func() {
		defer w.inFlight.Add(-1)
		timer := time.NewTimer(w.CreateDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			w.logger.Printf("phone=%s canceled before create: %v", phone, ctx.Err())
			return
		case <-timer.C:
		}

		w.createDelayedTask(ctx, phone)
	}()
}

func (w *Worker) createDelayedTask(ctx context.Context, phone string) {
	w.createMu.Lock()
	defer w.createMu.Unlock()

	pendingOthers := w.inFlight.Load() - 1
	if pendingOthers < 0 {
		pendingOthers = 0
	}
	idle, err := w.System.IdleDeviceCount(ctx)
	if err != nil {
		w.logger.Printf("phone=%s refresh idle before create failed: %v", phone, err)
	} else {
		capacity := idle - w.IdleThreshold - pendingOthers
		w.logger.Printf("phone=%s refreshed idle=%d threshold=%d pendingOthers=%d createCapacity=%d", phone, idle, w.IdleThreshold, pendingOthers, capacity)
	}
	taskID, err := w.System.CreateUserSentTask(ctx, phone)
	if err != nil {
		w.logger.Printf("create task phone=%s failed: %v", phone, err)
		return
	}
	w.logger.Printf("created task id=%d phone=%s mode=%s", taskID, phone, smsReceiveModeUserSent)
}

func (w *Worker) WaitIdle(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if w.inFlight.Load() <= 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
