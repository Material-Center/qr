package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"phone-task-client/internal/backend"
	"phone-task-client/internal/domain"
	"phone-task-client/internal/source"
	"phone-task-client/internal/store"
)

type fakeBackend struct {
	idleCalls int
	idle      int64
	nextID    uint
	created   []string
	tasks     map[uint]backend.TaskInfo
	submitted []uint
}

func (f *fakeBackend) IdleDeviceCount(ctx context.Context) (int64, error) {
	f.idleCalls++
	return f.idle, nil
}

func (f *fakeBackend) CreateSendCodeTask(ctx context.Context, phone string, createDelay time.Duration) (backend.TaskInfo, error) {
	f.nextID++
	f.created = append(f.created, phone)
	task := backend.TaskInfo{ID: f.nextID, Phone: phone, Status: "created", SMSReceiveMode: backend.SMSReceiveModeUserSent}
	f.tasks[task.ID] = task
	return task, nil
}

func (f *fakeBackend) CreateReceiveCodeTask(ctx context.Context, phone string, createDelay time.Duration) (backend.TaskInfo, error) {
	f.nextID++
	f.created = append(f.created, phone)
	task := backend.TaskInfo{ID: f.nextID, Phone: phone, Status: "waiting_promoter_code", NeedPromoterCode: true, SMSReceiveMode: backend.SMSReceiveModePlatformSend}
	f.tasks[task.ID] = task
	return task, nil
}

func (f *fakeBackend) GetTask(ctx context.Context, taskID uint) (backend.TaskInfo, error) {
	task, ok := f.tasks[taskID]
	if !ok {
		return backend.TaskInfo{}, errors.New("missing task")
	}
	return task, nil
}

func (f *fakeBackend) SubmitCode(ctx context.Context, taskID uint, verifyCode string) (backend.TaskInfo, error) {
	f.submitted = append(f.submitted, taskID)
	task := f.tasks[taskID]
	task.Status = "running"
	f.tasks[taskID] = task
	return task, nil
}

type fakeSource struct {
	phones []string
	code   string
}

func (f *fakeSource) FetchPhone(ctx context.Context, tmpl domain.APITemplate, now time.Time) (string, error) {
	if len(f.phones) == 0 {
		return "", source.ErrPhoneNotReady
	}
	phone := f.phones[0]
	f.phones = f.phones[1:]
	return phone, nil
}

func (f *fakeSource) FetchCode(ctx context.Context, tmpl domain.APITemplate, phone string, now time.Time) (string, source.VerifyCodeDetail, error) {
	return f.code, source.VerifyCodeDetail{Source: "text"}, nil
}

func newRunnerTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "client.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.SaveGlobalSettings(domain.GlobalSettings{
		BaseURL:        "https://server.test",
		ReserveDevices: 1,
		Interval:       3 * time.Second,
		Timeout:        10 * time.Second,
		LogDir:         "logs",
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	return st
}

func TestRunnerRunOnceUsesOneIdleQueryForMultipleJobs(t *testing.T) {
	st := newRunnerTestStore(t)
	now := time.Unix(100, 0)
	for i, phone := range []string{"18507561351", "18507561352"} {
		_, _, err := st.CreateJob(domain.Job{
			Name:            "send",
			ProfileID:       int64(i + 1),
			TaskType:        domain.TaskTypeSendCode,
			PhoneSourceType: domain.SourceTypeTXT,
			CodeSourceType:  domain.SourceTypeNone,
			BaseURLSnapshot: "https://server.test",
			Status:          domain.JobStatusRunning,
			CreatedAt:       now.Add(time.Duration(i) * time.Second),
			UpdatedAt:       now.Add(time.Duration(i) * time.Second),
		}, []domain.JobItem{{Phone: phone, Status: domain.JobItemStatusPending}})
		if err != nil {
			t.Fatalf("create job: %v", err)
		}
	}
	fb := &fakeBackend{idle: 3, tasks: map[uint]backend.TaskInfo{}}
	runner := NewRunner(st, fb, &fakeSource{}, func() time.Time { return now })

	if err := runner.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if fb.idleCalls != 1 {
		t.Fatalf("idle calls = %d", fb.idleCalls)
	}
	if len(fb.created) != 2 {
		t.Fatalf("created = %#v", fb.created)
	}
}

func TestRunnerReceiveCodeCreatesFetchesAndSubmitsCode(t *testing.T) {
	st := newRunnerTestStore(t)
	now := time.Unix(100, 0)
	job, _, err := st.CreateJob(domain.Job{
		Name:            "receive",
		ProfileID:       1,
		TaskType:        domain.TaskTypeReceiveCode,
		PhoneSourceType: domain.SourceTypeTXT,
		CodeSourceType:  domain.SourceTypeAPI,
		BaseURLSnapshot: "https://server.test",
		Status:          domain.JobStatusRunning,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, []domain.JobItem{{Phone: "13238381229", Status: domain.JobItemStatusPending, SourceLineNo: 2}})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	fb := &fakeBackend{idle: 2, tasks: map[uint]backend.TaskInfo{}}
	runner := NewRunner(st, fb, &fakeSource{code: "220220"}, func() time.Time { return now })

	if err := runner.RunOnce(t.Context()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	items, err := st.ListJobItems(job.ID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(fb.submitted) != 1 {
		t.Fatalf("submitted = %#v", fb.submitted)
	}
	if items[0].Status != domain.JobItemStatusSucceeded || items[0].VerifyCode != "220220" {
		t.Fatalf("items = %#v", items)
	}
}
