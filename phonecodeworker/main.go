package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const defaultSystemBaseURL = "http://210.16.170.132:1111/api"

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

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
		successPath   = flag.String("success-output", "", "succeeded import output path; default is <input>.success.txt")
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
	if *successPath == "" {
		*successPath = *input + ".success.txt"
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
	logger.Printf("startup settings %s", formatStartupSettings(startupSettings{
		Version:       version,
		GitCommit:     gitCommit,
		BuildTime:     buildTime,
		BaseURL:       *baseURL,
		Token:         *token,
		Input:         *input,
		StatePath:     *statePath,
		FailedPath:    *failedPath,
		SuccessPath:   *successPath,
		PauseFile:     *pauseFile,
		LogDir:        *logDir,
		CodeAPI:       state.CodeAPI,
		PhoneCount:    len(state.Records),
		Interval:      *interval,
		IdleThreshold: *idleThreshold,
		CreateDelay:   *createDelay,
		TaskSyncLimit: *taskSyncLimit,
		Timeout:       *timeout,
		Once:          *once,
	}))
	logger.Printf("loaded import input=%s state=%s failedOutput=%s successOutput=%s phones=%d baseURL=%s interval=%s idleThreshold=%d createDelay=%s taskSyncLimit=%d timeout=%s logDir=%s once=%t",
		*input, *statePath, *failedPath, *successPath, len(state.Records), *baseURL, *interval, *idleThreshold, *createDelay, *taskSyncLimit, *timeout, *logDir, *once)
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(*baseURL, *token, *timeout),
		CodeSource:    NewCodeSourceClient(state.CodeAPI, *timeout),
		State:         state,
		StatePath:     *statePath,
		FailedPath:    *failedPath,
		SuccessPath:   *successPath,
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

type startupSettings struct {
	Version       string
	GitCommit     string
	BuildTime     string
	BaseURL       string
	Token         string
	Input         string
	StatePath     string
	FailedPath    string
	SuccessPath   string
	PauseFile     string
	LogDir        string
	CodeAPI       string
	PhoneCount    int
	Interval      time.Duration
	IdleThreshold int64
	CreateDelay   time.Duration
	TaskSyncLimit int
	Timeout       time.Duration
	Once          bool
}

func formatStartupSettings(s startupSettings) string {
	return fmt.Sprintf(
		"version=%s gitCommit=%s buildTime=%s baseURL=%s token=%s input=%s state=%s failedOutput=%s successOutput=%s pauseFile=%s logDir=%s codeAPI=%s phones=%d interval=%s idleThreshold=%d createDelay=%s taskSyncLimit=%d timeout=%s once=%t",
		strings.TrimSpace(s.Version),
		strings.TrimSpace(s.GitCommit),
		strings.TrimSpace(s.BuildTime),
		strings.TrimSpace(s.BaseURL),
		maskStartupSecret(s.Token),
		strings.TrimSpace(s.Input),
		strings.TrimSpace(s.StatePath),
		strings.TrimSpace(s.FailedPath),
		strings.TrimSpace(s.SuccessPath),
		strings.TrimSpace(s.PauseFile),
		strings.TrimSpace(s.LogDir),
		sanitizeStartupURL(s.CodeAPI),
		s.PhoneCount,
		s.Interval,
		s.IdleThreshold,
		s.CreateDelay,
		s.TaskSyncLimit,
		s.Timeout,
		s.Once,
	)
}

func maskStartupSecret(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 8 {
		return strings.Repeat("*", len(raw))
	}
	return raw[:2] + strings.Repeat("*", len(raw)-5) + raw[len(raw)-3:]
}

func sanitizeStartupURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if idx := strings.Index(raw, "?"); idx >= 0 {
			return raw[:idx] + "?***"
		}
		return raw
	}
	query := parsed.Query()
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for key := range query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		masked := url.Values{}
		for _, key := range keys {
			values := query[key]
			if len(values) == 0 {
				masked[key] = []string{""}
				continue
			}
			for _, value := range values {
				if value == "" {
					masked.Add(key, "")
				} else {
					masked.Add(key, "***")
				}
			}
		}
		parsed.RawQuery = masked.Encode()
	}
	return strings.ReplaceAll(parsed.String(), "%2A%2A%2A", "***")
}
