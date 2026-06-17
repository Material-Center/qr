package core

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"phone-task-client/internal/backend"
	"phone-task-client/internal/domain"
	"phone-task-client/internal/source"
	"phone-task-client/internal/store"
)

type Backend interface {
	IdleDeviceCount(ctx context.Context) (int64, error)
	CreateSendCodeTask(ctx context.Context, phone string, createDelay time.Duration) (backend.TaskInfo, error)
	CreateReceiveCodeTask(ctx context.Context, phone string, createDelay time.Duration) (backend.TaskInfo, error)
	GetTask(ctx context.Context, taskID uint) (backend.TaskInfo, error)
	SubmitCode(ctx context.Context, taskID uint, verifyCode string) (backend.TaskInfo, error)
}

type SourceClient interface {
	FetchPhone(ctx context.Context, tmpl domain.APITemplate, now time.Time) (string, error)
	FetchCode(ctx context.Context, tmpl domain.APITemplate, phone string, now time.Time) (string, source.VerifyCodeDetail, error)
}

type Runner struct {
	store   *store.Store
	backend Backend
	source  SourceClient
	now     func() time.Time
}

func NewRunner(store *store.Store, backend Backend, source SourceClient, now func() time.Time) *Runner {
	if now == nil {
		now = time.Now
	}
	return &Runner{store: store, backend: backend, source: source, now: now}
}

func (r *Runner) RunOnce(ctx context.Context) error {
	jobs, err := r.store.ListRunnableJobs()
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}
	if err := r.syncActiveItems(ctx, jobs); err != nil {
		return err
	}

	settings, err := r.store.LoadGlobalSettings()
	if err != nil {
		return err
	}
	idle, err := r.backend.IdleDeviceCount(ctx)
	if err != nil {
		return err
	}
	snapshot := BuildDevicePoolSnapshot(settings.BaseURL, idle, settings.ReserveDevices, countProfiles(jobs), len(jobs), 0, r.now(), "")
	pendingJobs, err := r.pendingJobs(jobs, snapshot.Capacity)
	if err != nil {
		return err
	}
	allocations := AllocateSharedCapacity(snapshot.Capacity, pendingJobs)
	for _, allocation := range allocations {
		job := findJob(jobs, allocation.JobID)
		if job.ID == 0 {
			continue
		}
		if err := r.createForJob(ctx, job, allocation.Slots); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) syncActiveItems(ctx context.Context, jobs []domain.Job) error {
	for _, job := range jobs {
		items, err := r.store.ListItemsByStatus(job.ID, 0,
			domain.JobItemStatusCreated,
			domain.JobItemStatusWaitingCode,
			domain.JobItemStatusCodeSubmitted,
		)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.RemoteTaskID == 0 {
				continue
			}
			task, err := r.backend.GetTask(ctx, item.RemoteTaskID)
			if err != nil {
				item.LastError = err.Error()
				item.UpdatedAt = r.now()
				if saveErr := r.store.UpdateJobItem(item); saveErr != nil {
					return saveErr
				}
				continue
			}
			if err := r.handleTask(ctx, job, item, task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) pendingJobs(jobs []domain.Job, capacity int64) ([]PendingJob, error) {
	out := make([]PendingJob, 0, len(jobs))
	for _, job := range jobs {
		pending := 0
		if job.PhoneSourceType == domain.SourceTypeAPI {
			pending = int(capacity)
		} else {
			count, err := r.store.CountItemsByStatus(job.ID, domain.JobItemStatusPending)
			if err != nil {
				return nil, err
			}
			pending = count
		}
		if pending <= 0 {
			continue
		}
		out = append(out, PendingJob{
			JobID:        job.ID,
			ProfileID:    job.ProfileID,
			PendingItems: pending,
			UpdatedAt:    job.UpdatedAt,
		})
	}
	return out, nil
}

func (r *Runner) createForJob(ctx context.Context, job domain.Job, slots int) error {
	for i := 0; i < slots; i++ {
		item, ok, err := r.nextItem(ctx, job)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		switch job.TaskType {
		case domain.TaskTypeSendCode:
			task, err := r.backend.CreateSendCodeTask(ctx, item.Phone, job.CreateDelaySnapshot)
			if err != nil {
				item.LastError = err.Error()
				item.UpdatedAt = r.now()
				if saveErr := r.store.UpdateJobItem(item); saveErr != nil {
					return saveErr
				}
				continue
			}
			item.RemoteTaskID = task.ID
			item.RemoteStatus = task.Status
			item.Status = domain.JobItemStatusSucceeded
			item.UpdatedAt = r.now()
			if err := r.store.UpdateJobItem(item); err != nil {
				return err
			}
		case domain.TaskTypeReceiveCode:
			task, err := r.backend.CreateReceiveCodeTask(ctx, item.Phone, job.CreateDelaySnapshot)
			if err != nil {
				item.LastError = err.Error()
				item.UpdatedAt = r.now()
				if saveErr := r.store.UpdateJobItem(item); saveErr != nil {
					return saveErr
				}
				continue
			}
			item.RemoteTaskID = task.ID
			item.RemoteStatus = task.Status
			item.Status = domain.JobItemStatusCreated
			item.UpdatedAt = r.now()
			if err := r.handleTask(ctx, job, item, task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) nextItem(ctx context.Context, job domain.Job) (domain.JobItem, bool, error) {
	if job.PhoneSourceType == domain.SourceTypeAPI {
		tmpl := decodeAPITemplate(job.PhoneSourceConfigJSON)
		phone, err := r.source.FetchPhone(ctx, tmpl, r.now())
		if err != nil {
			if errors.Is(err, source.ErrPhoneNotReady) {
				return domain.JobItem{}, false, nil
			}
			return domain.JobItem{}, false, err
		}
		item, err := r.store.AddJobItem(domain.JobItem{
			JobID:     job.ID,
			Phone:     phone,
			Status:    domain.JobItemStatusPending,
			CreatedAt: r.now(),
			UpdatedAt: r.now(),
		})
		return item, err == nil, err
	}
	items, err := r.store.ListItemsByStatus(job.ID, 1, domain.JobItemStatusPending)
	if err != nil {
		return domain.JobItem{}, false, err
	}
	if len(items) == 0 {
		return domain.JobItem{}, false, nil
	}
	return items[0], true, nil
}

func (r *Runner) handleTask(ctx context.Context, job domain.Job, item domain.JobItem, task backend.TaskInfo) error {
	item.RemoteStatus = task.Status
	if task.Status == "failed" || task.FinishedAt != nil && task.Status != "succeeded" {
		item.Status = domain.JobItemStatusFailed
		item.LastError = task.LastError
		item.UpdatedAt = r.now()
		return r.store.UpdateJobItem(item)
	}
	if task.Status == "succeeded" {
		item.Status = domain.JobItemStatusSucceeded
		item.LastError = ""
		item.UpdatedAt = r.now()
		return r.store.UpdateJobItem(item)
	}
	if job.TaskType != domain.TaskTypeReceiveCode {
		item.UpdatedAt = r.now()
		return r.store.UpdateJobItem(item)
	}
	if task.NeedPromoterCode || task.Status == "waiting_promoter_code" {
		tmpl := decodeAPITemplate(job.CodeSourceConfigJSON)
		code, _, err := r.source.FetchCode(ctx, tmpl, item.Phone, r.now())
		if err != nil {
			if errors.Is(err, source.ErrCodeNotReady) {
				item.Status = domain.JobItemStatusWaitingCode
				item.LastError = "验证码未就绪"
				item.UpdatedAt = r.now()
				return r.store.UpdateJobItem(item)
			}
			item.LastError = err.Error()
			item.UpdatedAt = r.now()
			return r.store.UpdateJobItem(item)
		}
		submittedTask, err := r.backend.SubmitCode(ctx, item.RemoteTaskID, code)
		if err != nil {
			item.LastError = err.Error()
			item.UpdatedAt = r.now()
			return r.store.UpdateJobItem(item)
		}
		item.VerifyCode = code
		item.RemoteStatus = submittedTask.Status
		item.Status = domain.JobItemStatusSucceeded
		item.LastError = ""
		item.UpdatedAt = r.now()
		return r.store.UpdateJobItem(item)
	}
	item.Status = domain.JobItemStatusCreated
	item.UpdatedAt = r.now()
	return r.store.UpdateJobItem(item)
}

func decodeAPITemplate(raw string) domain.APITemplate {
	var tmpl domain.APITemplate
	_ = json.Unmarshal([]byte(raw), &tmpl)
	return tmpl
}

func countProfiles(jobs []domain.Job) int {
	seen := map[int64]struct{}{}
	for _, job := range jobs {
		seen[job.ProfileID] = struct{}{}
	}
	return len(seen)
}

func findJob(jobs []domain.Job, id int64) domain.Job {
	for _, job := range jobs {
		if job.ID == id {
			return job
		}
	}
	return domain.Job{}
}
