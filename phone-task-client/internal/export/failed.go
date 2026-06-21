package export

import (
	"encoding/json"
	"net/url"
	"strings"

	"phone-task-client/internal/domain"
)

func BuildFailedRetryFile(job domain.Job, items []domain.JobItem) (string, error) {
	return buildStatusFile(job, items, domain.JobItemStatusFailed)
}

func BuildSucceededFile(job domain.Job, items []domain.JobItem) (string, error) {
	return buildStatusFile(job, items, domain.JobItemStatusSucceeded)
}

func buildStatusFile(job domain.Job, items []domain.JobItem, status domain.JobItemStatus) (string, error) {
	phones := []string{}
	for _, item := range items {
		if item.Status != status {
			continue
		}
		phone := strings.TrimSpace(item.Phone)
		if phone == "" {
			continue
		}
		phones = append(phones, phone)
	}
	if len(phones) == 0 {
		return "", nil
	}

	var builder strings.Builder
	if job.TaskType == domain.TaskTypeReceiveCode {
		codeAPI := retryCodeAPI(job.CodeSourceConfigJSON)
		if codeAPI != "" {
			builder.WriteString(codeAPI)
			builder.WriteByte('\n')
		}
	}
	for _, phone := range phones {
		builder.WriteString(phone)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}

func retryCodeAPI(raw string) string {
	var tmpl domain.APITemplate
	if err := json.Unmarshal([]byte(raw), &tmpl); err != nil {
		return ""
	}
	if strings.Contains(tmpl.URL, "{phone}") {
		return strings.TrimSpace(tmpl.URL)
	}
	parsed, err := url.Parse(strings.TrimSpace(tmpl.URL))
	if err != nil {
		return strings.TrimSpace(tmpl.URL)
	}
	query := parsed.Query()
	for key, value := range tmpl.Query {
		query.Set(key, value)
	}
	if _, ok := query["phone"]; !ok {
		query.Set("phone", "{phone}")
	}
	parsed.RawQuery = strings.ReplaceAll(query.Encode(), "%7Bphone%7D", "{phone}")
	return parsed.String()
}
