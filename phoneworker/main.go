package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const defaultPhoneSourceURL = "http://206.238.179.123:37520/OPenApi/GetOrder?infor=vwZt5p4FmyeupCqMqKsC2ktcpczoBuX23akOGMEPlsw%3D&project=wb3"
const defaultSystemBaseURL = "http://210.16.170.132:1111/api"

func main() {
	if err := run(); err != nil {
		log.Printf("exit: %v", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		baseURL       = flag.String("base-url", defaultSystemBaseURL, "system API base URL")
		token         = flag.String("token", "", "promoter OpenAPI token")
		phoneURL      = flag.String("phone-url", defaultPhoneSourceURL, "phone source API URL")
		pauseFile     = flag.String("pause-file", defaultPauseFile("phoneworker.pause"), "pause control file path; when present, no new tasks are created")
		logDir        = flag.String("log-dir", defaultLogDir(), "log directory; a new log file is created on startup and rotated daily")
		interval      = flag.Duration("interval", 3*time.Second, "poll interval")
		idleThreshold = flag.Int64("idle-threshold", 1, "create a task only when idle device count is greater than this value")
		createDelay   = flag.Duration("create-delay", 0, "server-side delay before task can be claimed")
		timeout       = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
		once          = flag.Bool("once", false, "run one cycle and exit")
	)
	flag.Parse()

	if *token == "" {
		return fmt.Errorf("-token is required")
	}
	sourceURL, err := buildPhoneSourceURL(*phoneURL)
	if err != nil {
		return fmt.Errorf("invalid -phone-url: %w", err)
	}

	logWriter, err := newDailyLogWriter(*logDir, "phoneworker", time.Now)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logWriter.Close()
	logger := log.New(io.MultiWriter(os.Stdout, logWriter), "", log.LstdFlags)
	logger.Printf("log file=%s", logWriter.Path())
	logger.Printf("loaded config baseURL=%s phoneURL=%s interval=%s idleThreshold=%d createDelay=%s timeout=%s logDir=%s once=%t",
		*baseURL, sourceURL, *interval, *idleThreshold, *createDelay, *timeout, *logDir, *once)
	system := NewSystemClient(*baseURL, *token, *timeout)
	source := NewPhoneSourceClient(sourceURL, *timeout)
	worker := NewWorker(workerConfig{
		System:        system,
		PhoneSource:   source,
		PauseFile:     *pauseFile,
		IdleThreshold: *idleThreshold,
		Interval:      *interval,
		CreateDelay:   *createDelay,
		Logger:        logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *once {
		if err := worker.RunOnce(ctx); err != nil {
			return err
		}
		return worker.WaitIdle(ctx)
	}
	return worker.Run(ctx)
}

func defaultPauseFile(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return name
	}
	return filepath.Join(filepath.Dir(exe), name)
}
