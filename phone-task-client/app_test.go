package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"phone-task-client/internal/domain"
	"phone-task-client/internal/store"
)

func TestEnsureDefaultAPITemplatesSeedsSendAndReceiveDefaults(t *testing.T) {
	st := newAppTestStore(t)

	if err := ensureDefaultAPITemplates(st); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	items, err := st.ListAPITemplates()
	if err != nil {
		t.Fatalf("list api templates: %v", err)
	}
	if !hasAPITemplate(items, domain.APITypePhoneSource, defaultPhoneSourceURL) {
		t.Fatalf("missing default phone source template: %#v", items)
	}
	if !hasAPITemplate(items, domain.APITypeCodeSource, defaultCodeSourceURL) {
		t.Fatalf("missing default code source template: %#v", items)
	}
}

func TestEnsureDefaultAPITemplatesIsIdempotent(t *testing.T) {
	st := newAppTestStore(t)

	if err := ensureDefaultAPITemplates(st); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	if err := ensureDefaultAPITemplates(st); err != nil {
		t.Fatalf("seed defaults again: %v", err)
	}

	items, err := st.ListAPITemplates()
	if err != nil {
		t.Fatalf("list api templates: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("template count = %d, want 2: %#v", len(items), items)
	}
}

func TestStatusIncludesBuildVersion(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	oldBuildTime := buildTime
	version = "phone-task-client"
	gitCommit = "abc1234"
	buildTime = "2026-06-19T00:00:00Z"
	t.Cleanup(func() {
		version = oldVersion
		gitCommit = oldGitCommit
		buildTime = oldBuildTime
	})

	status := NewApp().Status()
	if status.Version != "phone-task-client" {
		t.Fatalf("version = %q", status.Version)
	}
	if status.GitCommit != "abc1234" {
		t.Fatalf("git commit = %q", status.GitCommit)
	}
	if status.BuildTime != "2026-06-19T00:00:00Z" {
		t.Fatalf("build time = %q", status.BuildTime)
	}
}

func TestDefaultDBPathUsesExecutableDataDirectory(t *testing.T) {
	oldExecutablePath := executablePath
	exeDir := t.TempDir()
	executablePath = func() (string, error) {
		return filepath.Join(exeDir, "phone-task-client-ui.exe"), nil
	}
	t.Cleanup(func() {
		executablePath = oldExecutablePath
	})

	got := defaultDBPath()
	if filepath.Base(got) != "phone-task-client.db" {
		t.Fatalf("db filename = %q", filepath.Base(got))
	}
	if filepath.Base(filepath.Dir(got)) != "data" {
		t.Fatalf("db path should use executable data directory, got %q", got)
	}
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Fatalf("data directory should be created: %v", err)
	}
}

func TestDefaultDBPathUsesDevDirectoryWhenConfigured(t *testing.T) {
	devDir := t.TempDir()
	t.Setenv("PHONE_TASK_CLIENT_DEV_DIR", devDir)

	got := defaultDBPath()
	want := filepath.Join(devDir, "data", "phone-task-client.db")
	if got != want {
		t.Fatalf("db path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Fatalf("dev data directory should be created: %v", err)
	}
}

func TestDefaultLogDirUsesExecutableLogsDirectory(t *testing.T) {
	oldExecutablePath := executablePath
	exeDir := t.TempDir()
	executablePath = func() (string, error) {
		return filepath.Join(exeDir, "phone-task-client-ui.exe"), nil
	}
	t.Cleanup(func() {
		executablePath = oldExecutablePath
	})

	got := defaultLogDir()
	if filepath.Base(got) != "logs" {
		t.Fatalf("log directory should be logs, got %q", got)
	}
	if filepath.Dir(got) != exeDir {
		t.Fatalf("log directory should be under executable directory, got %q", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("logs directory should be created: %v", err)
	}
}

func TestDefaultLogDirUsesDevDirectoryWhenConfigured(t *testing.T) {
	devDir := t.TempDir()
	t.Setenv("PHONE_TASK_CLIENT_DEV_DIR", devDir)

	got := defaultLogDir()
	want := filepath.Join(devDir, "logs")
	if got != want {
		t.Fatalf("log directory = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("dev logs directory should be created: %v", err)
	}
}

func TestStartJobCreatesPendingJobForManualRun(t *testing.T) {
	st := newAppTestStore(t)
	if err := st.SaveGlobalSettings(domain.GlobalSettings{
		BaseURL:  defaultSystemBaseURL,
		Interval: 3 * time.Second,
		Timeout:  time.Second,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	profile, err := st.SaveProfile(domain.Profile{
		Name:     "sales-1",
		TokenRef: "openapi-token",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}
	codeAPI, err := st.SaveAPITemplate(domain.APITemplate{
		Name:         "code-api",
		APIType:      domain.APITypeCodeSource,
		Method:       domain.HTTPMethodGET,
		URL:          "https://code.test/?phone={phone}",
		ResponseType: domain.ResponseTypeAuto,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("save code api: %v", err)
	}
	input := filepath.Join(t.TempDir(), "phones.txt")
	if err := os.WriteFile(input, []byte("18507561351\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	app := NewApp()
	app.store = st

	job, err := app.StartJob(StartJobRequest{
		Name:              "manual-start",
		ProfileID:         profile.ID,
		TaskType:          string(domain.TaskTypeReceiveCode),
		PhoneSourceType:   string(domain.SourceTypeTXT),
		CodeAPITemplateID: codeAPI.ID,
		InputPath:         input,
	})
	if err != nil {
		t.Fatalf("start job: %v", err)
	}
	if job.Status != domain.JobStatusPending {
		t.Fatalf("job status = %s, want %s", job.Status, domain.JobStatusPending)
	}
	runnable, err := st.ListRunnableJobs()
	if err != nil {
		t.Fatalf("list runnable jobs: %v", err)
	}
	if len(runnable) != 0 {
		t.Fatalf("runnable jobs = %d, want 0", len(runnable))
	}
}

func TestDashboardIncludesDeviceStats(t *testing.T) {
	st := newAppTestStore(t)
	if err := st.SaveGlobalSettings(domain.GlobalSettings{
		BaseURL:        defaultSystemBaseURL,
		ReserveDevices: 1,
		Interval:       3 * time.Second,
		Timeout:        time.Second,
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := st.SaveProfile(domain.Profile{
		Name:     "sales-1",
		TokenRef: "openapi-token",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	app := NewApp()
	app.store = st
	app.deviceStatsLoader = func(settings domain.GlobalSettings, profiles []domain.Profile) UIDeviceStats {
		if settings.ReserveDevices != 1 {
			t.Fatalf("reserve devices = %d", settings.ReserveDevices)
		}
		if len(profiles) != 1 || profiles[0].TokenRef != "openapi-token" {
			t.Fatalf("profiles = %#v", profiles)
		}
		return UIDeviceStats{Online: 5, Idle: 3, Reserve: settings.ReserveDevices, Capacity: 2}
	}

	dashboard, err := app.Dashboard()
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.DeviceStats.Online != 5 || dashboard.DeviceStats.Idle != 3 || dashboard.DeviceStats.Capacity != 2 {
		t.Fatalf("device stats = %#v", dashboard.DeviceStats)
	}
	if dashboard.DeviceStats.LastError != "" {
		t.Fatalf("device stats last error = %q", dashboard.DeviceStats.LastError)
	}
}

func TestListJobsPageReturnsSummariesAndTotal(t *testing.T) {
	st := newAppTestStore(t)
	now := time.Unix(100, 0)
	for i := 0; i < 4; i++ {
		_, _, err := st.CreateJob(domain.Job{
			Name:            "job",
			ProfileID:       1,
			TaskType:        domain.TaskTypeReceiveCode,
			PhoneSourceType: domain.SourceTypeTXT,
			Status:          domain.JobStatusPending,
			CreatedAt:       now.Add(time.Duration(i) * time.Second),
			UpdatedAt:       now.Add(time.Duration(i) * time.Second),
		}, []domain.JobItem{
			{Phone: "18507561351", Status: domain.JobItemStatusPending},
		})
		if err != nil {
			t.Fatalf("create job %d: %v", i, err)
		}
	}
	app := NewApp()
	app.store = st

	page, err := app.ListJobsPage(2, 2)
	if err != nil {
		t.Fatalf("list jobs page: %v", err)
	}
	if page.Total != 4 || page.Page != 2 || page.PageSize != 2 {
		t.Fatalf("page metadata = %#v", page)
	}
	if len(page.Jobs) != 2 {
		t.Fatalf("jobs len = %d, want 2", len(page.Jobs))
	}
	if page.Jobs[0].Job.ID != 2 || page.Jobs[1].Job.ID != 1 {
		t.Fatalf("job ids = %d,%d want 2,1", page.Jobs[0].Job.ID, page.Jobs[1].Job.ID)
	}
	if page.Jobs[0].Pending != 1 {
		t.Fatalf("job summary = %#v", page.Jobs[0])
	}
}

func TestExportSucceededWritesSucceededPhones(t *testing.T) {
	st := newAppTestStore(t)
	job, _, err := st.CreateJob(domain.Job{
		Name:                 "receive",
		TaskType:             domain.TaskTypeReceiveCode,
		PhoneSourceType:      domain.SourceTypeTXT,
		CodeSourceType:       domain.SourceTypeAPI,
		CodeSourceConfigJSON: `{"URL":"https://code.test/?phone={phone}","Method":"GET"}`,
		Status:               domain.JobStatusFinished,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}, []domain.JobItem{
		{Phone: "18507561351", Status: domain.JobItemStatusSucceeded},
		{Phone: "18507561352", Status: domain.JobItemStatusFailed},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	app := NewApp()
	app.store = st
	path := filepath.Join(t.TempDir(), "success.txt")

	if err := app.ExportSucceeded(job.ID, path); err != nil {
		t.Fatalf("export succeeded: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read succeeded file: %v", err)
	}
	if got, want := string(raw), "https://code.test/?phone={phone}\n18507561351\n"; got != want {
		t.Fatalf("succeeded file = %q, want %q", got, want)
	}
}

func TestDeleteJobRemovesJob(t *testing.T) {
	st := newAppTestStore(t)
	job, _, err := st.CreateJob(domain.Job{
		Name:            "delete-me",
		TaskType:        domain.TaskTypeReceiveCode,
		PhoneSourceType: domain.SourceTypeTXT,
		Status:          domain.JobStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	app := NewApp()
	app.store = st

	if err := app.DeleteJob(job.ID); err != nil {
		t.Fatalf("delete job: %v", err)
	}
	if _, err := st.GetJob(job.ID); err == nil {
		t.Fatal("get deleted job should fail")
	}
}

func newAppTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "client.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
