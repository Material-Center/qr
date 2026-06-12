package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	if *input == "" {
		return fmt.Errorf("-input is required")
	}
	if *statePath == "" {
		*statePath = *input + ".state.json"
	}

	data, err := LoadImportFile(*input)
	if err != nil {
		return err
	}
	state, err := LoadOrCreateState(*statePath, *input, data)
	if err != nil {
		return err
	}
	logger := log.New(os.Stdout, "", log.LstdFlags)
	worker := NewWorker(workerConfig{
		System:        NewSystemClient(*baseURL, *token, *timeout),
		CodeSource:    NewCodeSourceClient(state.CodeAPI, *timeout),
		State:         state,
		StatePath:     *statePath,
		IdleThreshold: *idleThreshold,
		Interval:      *interval,
		CreateDelay:   *createDelay,
		Logger:        logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *once {
		return worker.RunOnce(ctx)
	}
	return worker.Run(ctx)
}
