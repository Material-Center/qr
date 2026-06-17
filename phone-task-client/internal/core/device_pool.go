package core

import (
	"sort"
	"time"

	"phone-task-client/internal/domain"
)

const maxSlotsPerJobPerRound = 3

type PendingJob struct {
	JobID        int64
	ProfileID    int64
	PendingItems int
	UpdatedAt    time.Time
}

type JobAllocation struct {
	JobID int64
	Slots int
}

func BuildDevicePoolSnapshot(baseURL string, idle int64, reserve int64, runningProfiles int, runningJobs int, elapsed time.Duration, now time.Time, lastErr string) domain.DevicePoolSnapshot {
	capacity := idle - reserve
	if capacity < 0 {
		capacity = 0
	}
	return domain.DevicePoolSnapshot{
		BaseURL:             baseURL,
		IdleDeviceCount:     idle,
		ReserveDevices:      reserve,
		Capacity:            capacity,
		RunningProfileCount: runningProfiles,
		RunningJobCount:     runningJobs,
		QueryElapsed:        elapsed,
		LastError:           lastErr,
		CreatedAt:           now,
	}
}

func AllocateSharedCapacity(capacity int64, jobs []PendingJob) []JobAllocation {
	if capacity <= 0 {
		return nil
	}
	ordered := append([]PendingJob(nil), jobs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].UpdatedAt.Equal(ordered[j].UpdatedAt) {
			return ordered[i].JobID < ordered[j].JobID
		}
		return ordered[i].UpdatedAt.Before(ordered[j].UpdatedAt)
	})

	remaining := int(capacity)
	allocations := make([]JobAllocation, 0, len(ordered))
	for _, job := range ordered {
		if remaining <= 0 {
			break
		}
		if job.PendingItems <= 0 {
			continue
		}
		slots := minInt(job.PendingItems, maxSlotsPerJobPerRound, remaining)
		allocations = append(allocations, JobAllocation{JobID: job.JobID, Slots: slots})
		remaining -= slots
	}
	return allocations
}

func minInt(values ...int) int {
	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}
