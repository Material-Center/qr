package template

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"phone-task-client/internal/domain"
)

type RenderInput struct {
	Phone string
	Now   time.Time
}

func RenderGETURL(t domain.APITemplate, in RenderInput) (string, error) {
	if t.Method != "" && t.Method != domain.HTTPMethodGET {
		return "", fmt.Errorf("api template method must be GET")
	}
	if strings.TrimSpace(t.URL) == "" {
		return "", fmt.Errorf("api template url is required")
	}
	parsed, err := url.Parse(replaceVariables(t.URL, in))
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, value := range t.Query {
		query.Set(key, replaceVariables(value, in))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func replaceVariables(raw string, in RenderInput) string {
	out := strings.ReplaceAll(raw, "{phone}", strings.TrimSpace(in.Phone))
	if strings.Contains(out, "{timestamp}") {
		now := in.Now
		if now.IsZero() {
			now = time.Now()
		}
		out = strings.ReplaceAll(out, "{timestamp}", strconv.FormatInt(now.UnixMilli(), 10))
	}
	return out
}
