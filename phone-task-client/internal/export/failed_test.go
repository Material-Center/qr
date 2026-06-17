package export

import (
	"strings"
	"testing"

	"phone-task-client/internal/domain"
)

func TestBuildFailedRetryFileForReceiveCode(t *testing.T) {
	job := domain.Job{
		TaskType:             domain.TaskTypeReceiveCode,
		CodeSourceConfigJSON: `{"URL":"https://example.com/code","Query":{"phone":"{phone}"},"Method":"GET"}`,
	}
	items := []domain.JobItem{
		{Phone: "13238381229", Status: domain.JobItemStatusFailed},
		{Phone: "18507561351", Status: domain.JobItemStatusSucceeded},
		{Phone: "18507561352", Status: domain.JobItemStatusFailed},
	}

	got, err := BuildFailedRetryFile(job, items)
	if err != nil {
		t.Fatalf("build retry file: %v", err)
	}
	want := "https://example.com/code?phone={phone}\n13238381229\n18507561352\n"
	if got != want {
		t.Fatalf("retry file = %q, want %q", got, want)
	}
}

func TestBuildFailedRetryFileForSendCodeOnlyPhones(t *testing.T) {
	got, err := BuildFailedRetryFile(domain.Job{TaskType: domain.TaskTypeSendCode}, []domain.JobItem{
		{Phone: "13238381229", Status: domain.JobItemStatusFailed},
	})
	if err != nil {
		t.Fatalf("build retry file: %v", err)
	}
	if strings.TrimSpace(got) != "13238381229" {
		t.Fatalf("retry file = %q", got)
	}
}
