package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"phone-task-client/internal/domain"
	apitemplate "phone-task-client/internal/template"
)

var ErrCodeNotReady = errors.New("验证码未就绪")

type APIClient struct {
	http *http.Client
}

func NewAPIClient(timeout time.Duration) *APIClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &APIClient{http: &http.Client{Timeout: timeout}}
}

func (c *APIClient) FetchPhone(ctx context.Context, tmpl domain.APITemplate, now time.Time) (string, error) {
	requestURL, err := apitemplate.RenderGETURL(tmpl, apitemplate.RenderInput{Now: now})
	if err != nil {
		return "", err
	}
	body, err := c.get(ctx, requestURL)
	if err != nil {
		return "", err
	}
	return ExtractPhoneFromAPIResponse(body)
}

func (c *APIClient) FetchCode(ctx context.Context, tmpl domain.APITemplate, phone string, now time.Time) (string, VerifyCodeDetail, error) {
	requestURL, err := apitemplate.RenderGETURL(tmpl, apitemplate.RenderInput{Phone: phone, Now: now})
	if err != nil {
		return "", VerifyCodeDetail{}, err
	}
	body, err := c.get(ctx, requestURL)
	if err != nil {
		return "", VerifyCodeDetail{}, err
	}
	code, detail := ExtractVerifyCodeDetail(body)
	if code == "" {
		return "", detail, ErrCodeNotReady
	}
	return code, detail, nil
}

func (c *APIClient) get(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api request failed: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
