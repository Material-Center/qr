package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"phone-task-client/internal/backend"
	"phone-task-client/internal/core"
	"phone-task-client/internal/domain"
	phoneexport "phone-task-client/internal/export"
	"phone-task-client/internal/source"
	"phone-task-client/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Printf("exit: %v", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dbPath         = flag.String("db", "phone-task-client.db", "sqlite state database path")
		baseURL        = flag.String("base-url", "", "system API base URL")
		token          = flag.String("token", "", "promoter OpenAPI token")
		mode           = flag.String("mode", "receive", "task mode: send or receive")
		phoneSource    = flag.String("phone-source", "txt", "phone source: txt or api")
		input          = flag.String("input", "", "txt input file")
		phoneAPI       = flag.String("phone-api", "", "phone source API URL")
		codeAPI        = flag.String("code-api", "", "code API URL; receive mode only")
		jobID          = flag.Int64("job-id", 0, "resume existing job id")
		name           = flag.String("name", "", "job name")
		reserveDevices = flag.Int64("reserve-devices", 1, "global reserve device count")
		interval       = flag.Duration("interval", 3*time.Second, "poll interval")
		createDelay    = flag.Duration("create-delay", 0, "server-side create delay for this user")
		timeout        = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
		failedOutput   = flag.String("failed-output", "", "failed retry file output path")
		pauseJob       = flag.Int64("pause-job", 0, "pause an existing job id and exit")
		resumeJob      = flag.Int64("resume-job", 0, "resume an existing job id and exit")
		stopJob        = flag.Int64("stop-job", 0, "stop an existing job id and exit")
		once           = flag.Bool("once", false, "run one cycle and exit")
	)
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	if *pauseJob > 0 {
		return setJobControl(st, *pauseJob, "pause")
	}
	if *resumeJob > 0 {
		return setJobControl(st, *resumeJob, "resume")
	}
	if *stopJob > 0 {
		return setJobControl(st, *stopJob, "stop")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	settings := domain.GlobalSettings{
		BaseURL:        strings.TrimSpace(*baseURL),
		ReserveDevices: *reserveDevices,
		Interval:       *interval,
		Timeout:        *timeout,
		LogDir:         "logs",
	}
	if err := st.SaveGlobalSettings(settings); err != nil {
		return err
	}

	var job domain.Job
	if *jobID > 0 {
		job, err = st.GetJob(*jobID)
		if err != nil {
			return err
		}
	} else {
		job, err = createJobFromFlags(st, createJobOptions{
			BaseURL:        *baseURL,
			Token:          *token,
			Mode:           *mode,
			PhoneSource:    *phoneSource,
			Input:          *input,
			PhoneAPI:       *phoneAPI,
			CodeAPI:        *codeAPI,
			Name:           *name,
			ReserveDevices: *reserveDevices,
			Interval:       *interval,
			Timeout:        *timeout,
			CreateDelay:    *createDelay,
		})
		if err != nil {
			return err
		}
		log.Printf("created job id=%d name=%q", job.ID, job.Name)
	}

	system := backend.NewSystemClient(effectiveBaseURL(job, *baseURL), *token, *timeout)
	apiSource := source.NewAPIClient(*timeout)
	runner := core.NewRunner(st, system, apiSource, time.Now)

	for {
		if err := runner.RunOnce(ctx); err != nil {
			log.Printf("cycle error: %v", err)
		}
		if *once {
			break
		}
		done, err := jobDone(st, job.ID)
		if err != nil {
			return err
		}
		if done {
			break
		}
		select {
		case <-ctx.Done():
			log.Printf("stop requested, state saved db=%s job=%d", *dbPath, job.ID)
			return exportFailed(st, job.ID, *failedOutput)
		case <-time.After(*interval):
		}
	}
	return exportFailed(st, job.ID, *failedOutput)
}

type createJobOptions struct {
	BaseURL        string
	Token          string
	Mode           string
	PhoneSource    string
	Input          string
	PhoneAPI       string
	CodeAPI        string
	Name           string
	ReserveDevices int64
	Interval       time.Duration
	Timeout        time.Duration
	CreateDelay    time.Duration
}

func createJobFromFlags(st *store.Store, opts createJobOptions) (domain.Job, error) {
	if strings.TrimSpace(opts.BaseURL) == "" {
		return domain.Job{}, fmt.Errorf("-base-url is required")
	}
	if strings.TrimSpace(opts.Token) == "" {
		return domain.Job{}, fmt.Errorf("-token is required")
	}
	taskType, err := parseTaskType(opts.Mode)
	if err != nil {
		return domain.Job{}, err
	}
	sourceType, err := parseSourceType(opts.PhoneSource)
	if err != nil {
		return domain.Job{}, err
	}
	now := time.Now()
	if opts.Name == "" {
		opts.Name = fmt.Sprintf("%s-%s", taskType, now.Format("20060102-150405"))
	}

	profile, err := st.SaveProfile(domain.Profile{
		Name:        opts.Name,
		TokenRef:    opts.Token,
		TokenMask:   maskToken(opts.Token),
		CreateDelay: opts.CreateDelay,
		Enabled:     true,
	})
	if err != nil {
		return domain.Job{}, err
	}

	codeTemplate := domain.APITemplate{}
	phoneTemplate := domain.APITemplate{}
	var items []domain.JobItem
	codeSourceType := domain.SourceTypeNone
	if taskType == domain.TaskTypeReceiveCode {
		codeSourceType = domain.SourceTypeAPI
	}
	if sourceType == domain.SourceTypeTXT {
		if opts.Input == "" {
			return domain.Job{}, fmt.Errorf("-input is required for txt phone source")
		}
		raw, err := os.ReadFile(opts.Input)
		if err != nil {
			return domain.Job{}, err
		}
		firstLineCodeAPI := taskType == domain.TaskTypeReceiveCode && strings.TrimSpace(opts.CodeAPI) == ""
		parsed, err := source.ParseTXTImport(string(raw), firstLineCodeAPI)
		if err != nil {
			return domain.Job{}, err
		}
		if strings.TrimSpace(opts.CodeAPI) == "" {
			opts.CodeAPI = parsed.CodeAPI
		}
		for _, entry := range parsed.Phones {
			items = append(items, domain.JobItem{
				Phone:        entry.Phone,
				Status:       domain.JobItemStatusPending,
				SourceLineNo: entry.LineNo,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
	} else {
		if opts.PhoneAPI == "" {
			return domain.Job{}, fmt.Errorf("-phone-api is required for api phone source")
		}
		phoneTemplate = apiTemplateFromURL("phone-api", domain.APITypePhoneSource, opts.PhoneAPI, false)
	}
	if taskType == domain.TaskTypeReceiveCode {
		if strings.TrimSpace(opts.CodeAPI) == "" {
			return domain.Job{}, fmt.Errorf("-code-api is required for receive mode")
		}
		codeTemplate = apiTemplateFromURL("code-api", domain.APITypeCodeSource, opts.CodeAPI, true)
	}
	phoneConfig, _ := json.Marshal(phoneTemplate)
	codeConfig, _ := json.Marshal(codeTemplate)

	job := domain.Job{
		ProfileID:              profile.ID,
		Name:                   opts.Name,
		TaskType:               taskType,
		PhoneSourceType:        sourceType,
		CodeSourceType:         codeSourceType,
		PhoneSourceConfigJSON:  string(phoneConfig),
		CodeSourceConfigJSON:   string(codeConfig),
		BaseURLSnapshot:        strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		ReserveDevicesSnapshot: opts.ReserveDevices,
		IntervalSnapshot:       opts.Interval,
		TimeoutSnapshot:        opts.Timeout,
		CreateDelaySnapshot:    opts.CreateDelay,
		Status:                 domain.JobStatusRunning,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	job, _, err = st.CreateJob(job, items)
	return job, err
}

func apiTemplateFromURL(name string, apiType domain.APIType, rawURL string, includePhone bool) domain.APITemplate {
	tmpl := domain.APITemplate{
		Name:         name,
		APIType:      apiType,
		Method:       domain.HTTPMethodGET,
		URL:          strings.TrimSpace(rawURL),
		Query:        map[string]string{},
		ResponseType: domain.ResponseTypeAuto,
		Enabled:      true,
	}
	if includePhone && !strings.Contains(tmpl.URL, "{phone}") {
		tmpl.Query["phone"] = "{phone}"
	}
	return tmpl
}

func parseTaskType(raw string) (domain.TaskType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "send", "send_code":
		return domain.TaskTypeSendCode, nil
	case "receive", "receive_code":
		return domain.TaskTypeReceiveCode, nil
	default:
		return "", fmt.Errorf("unsupported mode %q", raw)
	}
}

func parseSourceType(raw string) (domain.SourceType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "txt":
		return domain.SourceTypeTXT, nil
	case "api":
		return domain.SourceTypeAPI, nil
	default:
		return "", fmt.Errorf("unsupported phone source %q", raw)
	}
}

func jobDone(st *store.Store, jobID int64) (bool, error) {
	pending, err := st.CountItemsByStatus(jobID, domain.JobItemStatusPending, domain.JobItemStatusCreated, domain.JobItemStatusWaitingCode, domain.JobItemStatusCodeSubmitted)
	if err != nil {
		return false, err
	}
	return pending == 0, nil
}

func exportFailed(st *store.Store, jobID int64, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	job, err := st.GetJob(jobID)
	if err != nil {
		return err
	}
	items, err := st.ListJobItems(jobID)
	if err != nil {
		return err
	}
	raw, err := phoneexport.BuildFailedRetryFile(job, items)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

func effectiveBaseURL(job domain.Job, fallback string) string {
	if strings.TrimSpace(job.BaseURLSnapshot) != "" {
		return job.BaseURLSnapshot
	}
	return fallback
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func setJobControl(st *store.Store, jobID int64, action string) error {
	job, err := st.GetJob(jobID)
	if err != nil {
		return err
	}
	switch action {
	case "pause":
		job.Paused = true
		job.Status = domain.JobStatusPaused
	case "resume":
		job.Paused = false
		job.Stopped = false
		job.Status = domain.JobStatusRunning
	case "stop":
		job.Stopped = true
		job.Status = domain.JobStatusStopped
	default:
		return fmt.Errorf("unsupported job action %q", action)
	}
	job.UpdatedAt = time.Now()
	if err := st.UpdateJob(job); err != nil {
		return err
	}
	log.Printf("job %d %s", jobID, action)
	return nil
}
