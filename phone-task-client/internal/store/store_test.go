package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"phone-task-client/internal/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "client.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}

func TestStoreGlobalSettingsRoundTrip(t *testing.T) {
	store := newTestStore(t)
	want := domain.GlobalSettings{
		BaseURL:        "https://server.test",
		ReserveDevices: 10,
		Interval:       3 * time.Second,
		Timeout:        10 * time.Second,
		LogDir:         "logs",
	}

	if err := store.SaveGlobalSettings(want); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	got, err := store.LoadGlobalSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if got != want {
		t.Fatalf("settings = %#v, want %#v", got, want)
	}
}

func TestStoreProfileAndTemplatesRoundTrip(t *testing.T) {
	store := newTestStore(t)
	profile, err := store.SaveProfile(domain.Profile{
		Name:            "sales-1",
		TokenRef:        "token-ref",
		TokenMask:       "abc****",
		BaseURLOverride: "https://override.test",
		CreateDelay:     2 * time.Second,
		Enabled:         true,
		Remark:          "remark",
	})
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if profile.ID == 0 {
		t.Fatalf("profile id = 0")
	}

	apiTemplate, err := store.SaveAPITemplate(domain.APITemplate{
		Name:         "code-api",
		APIType:      domain.APITypeCodeSource,
		Method:       domain.HTTPMethodGET,
		URL:          "https://example.com/code",
		Query:        map[string]string{"phone": "{phone}"},
		ResponseType: domain.ResponseTypeAuto,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("save api template: %v", err)
	}

	taskTemplate, err := store.SaveTaskTemplate(domain.TaskTemplate{
		Name:              "receive-txt",
		ProfileID:         profile.ID,
		TaskType:          domain.TaskTypeReceiveCode,
		PhoneSourceType:   domain.SourceTypeTXT,
		CodeSourceType:    domain.SourceTypeAPI,
		CodeAPITemplateID: apiTemplate.ID,
		FailedOutputDir:   "failed",
		Enabled:           true,
	})
	if err != nil {
		t.Fatalf("save task template: %v", err)
	}

	gotProfile, err := store.GetProfile(profile.ID)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if gotProfile.Name != "sales-1" || gotProfile.CreateDelay != 2*time.Second {
		t.Fatalf("profile = %#v", gotProfile)
	}
	gotAPI, err := store.GetAPITemplate(apiTemplate.ID)
	if err != nil {
		t.Fatalf("get api template: %v", err)
	}
	if gotAPI.Query["phone"] != "{phone}" {
		t.Fatalf("api template = %#v", gotAPI)
	}
	gotTask, err := store.GetTaskTemplate(taskTemplate.ID)
	if err != nil {
		t.Fatalf("get task template: %v", err)
	}
	if gotTask.ProfileID != profile.ID || gotTask.CodeAPITemplateID != apiTemplate.ID {
		t.Fatalf("task template = %#v", gotTask)
	}
}

func TestStoreCreateJobWithItemsAndUpdateItem(t *testing.T) {
	store := newTestStore(t)
	now := time.Unix(100, 0)
	job, items, err := store.CreateJob(domain.Job{
		Name:                   "batch-1",
		ProfileID:              1,
		TaskType:               domain.TaskTypeReceiveCode,
		PhoneSourceType:        domain.SourceTypeTXT,
		CodeSourceType:         domain.SourceTypeAPI,
		BaseURLSnapshot:        "https://server.test",
		ReserveDevicesSnapshot: 10,
		IntervalSnapshot:       3 * time.Second,
		TimeoutSnapshot:        10 * time.Second,
		CreateDelaySnapshot:    2 * time.Second,
		Status:                 domain.JobStatusRunning,
		CreatedAt:              now,
		UpdatedAt:              now,
	}, []domain.JobItem{
		{Phone: "13238381229", Status: domain.JobItemStatusPending, SourceLineNo: 2},
		{Phone: "18507561351", Status: domain.JobItemStatusPending, SourceLineNo: 3},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if job.ID == 0 || len(items) != 2 || items[0].ID == 0 {
		t.Fatalf("job=%#v items=%#v", job, items)
	}

	items[0].Status = domain.JobItemStatusCreated
	items[0].RemoteTaskID = 99
	items[0].UpdatedAt = now.Add(time.Second)
	if err := store.UpdateJobItem(items[0]); err != nil {
		t.Fatalf("update item: %v", err)
	}

	got, err := store.ListJobItems(job.ID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if got[0].Status != domain.JobItemStatusCreated || got[0].RemoteTaskID != 99 {
		t.Fatalf("items = %#v", got)
	}
}

func TestStoreListJobsPageReturnsTotalAndOffset(t *testing.T) {
	store := newTestStore(t)
	now := time.Unix(100, 0)
	for i := 0; i < 5; i++ {
		_, _, err := store.CreateJob(domain.Job{
			Name:            "job",
			ProfileID:       1,
			TaskType:        domain.TaskTypeReceiveCode,
			PhoneSourceType: domain.SourceTypeTXT,
			Status:          domain.JobStatusPending,
			CreatedAt:       now.Add(time.Duration(i) * time.Second),
			UpdatedAt:       now.Add(time.Duration(i) * time.Second),
		}, nil)
		if err != nil {
			t.Fatalf("create job %d: %v", i, err)
		}
	}

	jobs, total, err := store.ListJobsPage(2, 2)
	if err != nil {
		t.Fatalf("list jobs page: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs len = %d, want 2", len(jobs))
	}
	if jobs[0].ID != 3 || jobs[1].ID != 2 {
		t.Fatalf("paged jobs ids = %d,%d want 3,2", jobs[0].ID, jobs[1].ID)
	}
}

func TestStoreDeleteJobRemovesJobItemsAndEvents(t *testing.T) {
	store := newTestStore(t)
	job, items, err := store.CreateJob(domain.Job{
		Name:            "delete-me",
		ProfileID:       1,
		TaskType:        domain.TaskTypeReceiveCode,
		PhoneSourceType: domain.SourceTypeTXT,
		Status:          domain.JobStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, []domain.JobItem{
		{Phone: "18507561351", Status: domain.JobItemStatusPending},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := store.AddEvent(domain.Event{
		JobID:     job.ID,
		ItemID:    items[0].ID,
		Phone:     items[0].Phone,
		Level:     "info",
		EventType: "test",
		Message:   "created",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("add event: %v", err)
	}

	if err := store.DeleteJob(job.ID); err != nil {
		t.Fatalf("delete job: %v", err)
	}
	if _, err := store.GetJob(job.ID); err == nil {
		t.Fatal("get deleted job should fail")
	}
	gotItems, err := store.ListJobItems(job.ID)
	if err != nil {
		t.Fatalf("list deleted job items: %v", err)
	}
	if len(gotItems) != 0 {
		t.Fatalf("deleted job items = %#v", gotItems)
	}
}

func TestStoreDeleteJobRejectsRunningJob(t *testing.T) {
	store := newTestStore(t)
	job, items, err := store.CreateJob(domain.Job{
		Name:            "running",
		ProfileID:       1,
		TaskType:        domain.TaskTypeReceiveCode,
		PhoneSourceType: domain.SourceTypeTXT,
		Status:          domain.JobStatusRunning,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, []domain.JobItem{
		{Phone: "18507561351", Status: domain.JobItemStatusPending},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := store.DeleteJob(job.ID); err == nil {
		t.Fatal("delete running job should fail")
	} else if !strings.Contains(err.Error(), "请先停止") {
		t.Fatalf("delete running job error = %q", err.Error())
	}
	if _, err := store.GetJob(job.ID); err != nil {
		t.Fatalf("running job should remain: %v", err)
	}
	gotItems, err := store.ListJobItems(job.ID)
	if err != nil {
		t.Fatalf("list running job items: %v", err)
	}
	if len(gotItems) != len(items) {
		t.Fatalf("running job items = %d, want %d", len(gotItems), len(items))
	}
}
