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
		input         = flag.String("input", "", "import file path; first non-empty line is code API, remaining lines are phones")
		statePath     = flag.String("state", "", "state file path; default is <input>.state.json")
		failedPath    = flag.String("failed-output", "", "failed import output path; default is <input>.failed.txt")
		pauseFile     = flag.String("pause-file", defaultPauseFile("phonecodeworker.pause"), "pause control file path; when present, no new tasks are created")
		logDir        = flag.String("log-dir", defaultLogDir(), "log directory; a new log file is created on startup and rotated daily")
		interval      = flag.Duration("interval", 3*time.Second, "poll interval")
		idleThreshold = flag.Int64("idle-threshold", 1, "create a task only when idle device count is greater than this value")
		createDelay   = flag.Duration("create-delay", 0, "server-side delay before task can be claimed")
		taskSyncLimit = flag.Int("task-sync-limit", defaultTaskSyncLimit, "max active task status checks per cycle")
		timeout       = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
		once          = flag.Bool("once", false, "run one cycle and exit")
	)
	flag.Parse()

	if *token == "" {
		return fmt.Errorf("-token is required")
	}
	if *input == "" {
		return fmt.Errorf("-input is required")
	}
	if *statePath == "" {
		*statePath = *input + ".state.json"
	}
	if *failedPath == "" {
		*failedPath = *input + ".failed.txt"
	}

	data, err := LoadImportFile(*input)
	if err != nil {
		return err
	}
	state, err := LoadOrCreateState(*statePath, *input, data)
	if err != nil {
		return err
	}
	logWriter, err := newDailyLogWriter(*logDir, "phonecodeworker", time.Now)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logWriter.Close()
	logger := log.New(io.MultiWriter(os.Stdout, logWriter), "", log.LstdFlags)
	logger.Printf("log file=%s", logWriter.Path())
	logger.Printf("loaded import input=%s state=%s failedOutput=%s phones=%d baseURL=%s interval=%s idleThreshold=%d createDelay=%s taskSyncLimit=%d timeout=%s logDir=%s once=%t",
		*input, *statePath, *failedPath, len(state.Records), *baseURL, *interval, *idleThreshold, *createDelay, *taskSyncLimit, *timeout, *logDir, *once)
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(*baseURL, *token, *timeout),
		CodeSource:    NewCodeSourceClient(state.CodeAPI, *timeout),
		State:         state,
		StatePath:     *statePath,
		FailedPath:    *failedPath,
		PauseFile:     *pauseFile,
		IdleThreshold: *idleThreshold,
		Interval:      *interval,
		CreateDelay:   *createDelay,
		TaskSyncLimit: *taskSyncLimit,
		Logger:        logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *once {
		return worker.RunOnce(ctx)
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
