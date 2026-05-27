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
		interval      = flag.Duration("interval", 3*time.Second, "poll interval")
		idleThreshold = flag.Int64("idle-threshold", 1, "create a task only when idle device count is greater than this value")
		createDelay   = flag.Duration("create-delay", 0, "wait after fetching phone before creating task")
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

	logger := log.New(os.Stdout, "", log.LstdFlags)
	system := NewSystemClient(*baseURL, *token, *timeout)
	source := NewPhoneSourceClient(sourceURL, *timeout)
	worker := NewWorker(workerConfig{
		System:        system,
		PhoneSource:   source,
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
