package core

import (
	"testing"
	"time"
)

func TestBuildDevicePoolSnapshotComputesCapacity(t *testing.T) {
	now := time.Unix(100, 0)
	got := BuildDevicePoolSnapshot("https://server.test", 20, 3, 2, 4, 15*time.Millisecond, now, "")

	if got.BaseURL != "https://server.test" {
		t.Fatalf("base url = %q", got.BaseURL)
	}
	if got.Capacity != 17 {
		t.Fatalf("capacity = %d", got.Capacity)
	}
	if got.RunningProfileCount != 2 || got.RunningJobCount != 4 {
		t.Fatalf("running counts = %d/%d", got.RunningProfileCount, got.RunningJobCount)
	}
	if !got.CreatedAt.Equal(now) {
		t.Fatalf("created at = %s", got.CreatedAt)
	}
}

func TestBuildDevicePoolSnapshotClampsNegativeCapacity(t *testing.T) {
	got := BuildDevicePoolSnapshot("https://server.test", 1, 3, 1, 1, 0, time.Unix(100, 0), "")
	if got.Capacity != 0 {
		t.Fatalf("capacity = %d", got.Capacity)
	}
}

func TestAllocateSharedCapacityDoesNotExceedGlobalCapacity(t *testing.T) {
	jobs := []PendingJob{
		{JobID: 1, ProfileID: 10, PendingItems: 5, UpdatedAt: time.Unix(300, 0)},
		{JobID: 2, ProfileID: 11, PendingItems: 5, UpdatedAt: time.Unix(100, 0)},
		{JobID: 3, ProfileID: 10, PendingItems: 5, UpdatedAt: time.Unix(200, 0)},
	}

	got := AllocateSharedCapacity(4, jobs)

	if len(got) != 2 {
		t.Fatalf("allocations = %#v", got)
	}
	if got[0].JobID != 2 || got[0].Slots != 3 {
		t.Fatalf("first allocation = %#v", got[0])
	}
	if got[1].JobID != 3 || got[1].Slots != 1 {
		t.Fatalf("second allocation = %#v", got[1])
	}
	total := 0
	for _, alloc := range got {
		total += alloc.Slots
	}
	if total != 4 {
		t.Fatalf("total slots = %d", total)
	}
}

func TestAllocateSharedCapacitySkipsJobsWithoutPendingItems(t *testing.T) {
	jobs := []PendingJob{
		{JobID: 1, PendingItems: 0, UpdatedAt: time.Unix(100, 0)},
		{JobID: 2, PendingItems: 2, UpdatedAt: time.Unix(200, 0)},
	}

	got := AllocateSharedCapacity(3, jobs)

	if len(got) != 1 || got[0].JobID != 2 || got[0].Slots != 2 {
		t.Fatalf("allocations = %#v", got)
	}
}
