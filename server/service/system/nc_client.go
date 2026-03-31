package system

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	NCQueryTypeQL = 7
	NCQueryTypeCD = 39
	NCQueryTypeSF = 12
	NCQueryTypeDR = 55

	ncClientAPIURL      = "http://tx.wenben.cc/member/api.php"
	ncClientDownloadURL = "http://tx.wenben.cc/member/downloads.php"
	NCDefaultSpeed      = 1
	ncPollIntervalMs    = 2000
	ncDefaultWaitMs     = 240000
)

type NCClient struct {
	AppID      string
	Secret     string
	CKMd5      string
	HTTPClient *http.Client
}

type NCCreateOrderResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	PID  string `json:"pid,omitempty"`
}

type NCOrderDetail struct {
	PID    string `json:"pid"`
	Status string `json:"status"`
}

type NCQueryOrderResp struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data NCOrderDetail `json:"data"`
}

type NCDRResult struct {
	UIN            string
	PhoneOnlineDay int
	IsPhoneOnline  bool
	IsPcOnline     bool
	Status         string
	Columns        []string
	Raw            string
}

type NCQLResult struct {
	UIN     string
	Level   int
	Age     int
	Columns []string
	Raw     string
}

type NCRow struct {
	Columns []string
	Raw     string
}

func NewNCClient(appid, secret, ckMd5 string) *NCClient {
	return &NCClient{
		AppID:  strings.TrimSpace(appid),
		Secret: strings.TrimSpace(secret),
		CKMd5:  strings.TrimSpace(ckMd5),
		HTTPClient: &http.Client{
			Timeout: 240 * time.Second,
		},
	}
}

func (c *NCClient) NcQueryDR(uinList []string, waitMs int, speed int) ([]NCDRResult, error) {
	raw, _, err := c.NcQueryInfo(uinList, NCQueryTypeDR, waitMs, speed)
	if err != nil {
		return nil, err
	}
	return parseNCClientQueryDR(raw), nil
}

func (c *NCClient) NcQueryQL(uinList []string, waitMs int, speed int) ([]NCQLResult, error) {
	raw, _, err := c.NcQueryInfo(uinList, NCQueryTypeQL, waitMs, speed)
	if err != nil {
		return nil, err
	}
	return parseNCClientQueryQL(raw), nil
}

func (c *NCClient) NcQueryInfo(uinList []string, queryType int, waitMs int, speed int) (string, string, error) {
	if speed <= 0 {
		speed = NCDefaultSpeed
	}
	pid, err := c.CreateOrder(uinList, queryType, speed)
	if err != nil {
		return "", "", fmt.Errorf("create order failed: %w", err)
	}
	if _, err := c.WaitOrderDone(pid, waitMs); err != nil {
		return "", "", fmt.Errorf("wait order failed: %w", err)
	}
	text, err := c.GetOrderTextResult(queryType, pid)
	if err != nil {
		return "", "", fmt.Errorf("get order detail failed: %w", err)
	}
	if strings.Contains(text, "未处理完-请等待完成再下载") {
		return "", "", fmt.Errorf("order not done: %s", text)
	}
	return strings.TrimSpace(text), pid, nil
}

func (c *NCClient) CreateOrder(uinList []string, queryType int, speed int) (string, error) {
	form := url.Values{}
	form.Set("types", strconv.Itoa(queryType))
	form.Set("speed", strconv.Itoa(speed))
	form.Set("data", buildNCClientData(uinList))

	raw, err := c.callAPI(http.MethodPost, map[string]string{}, form)
	if err != nil {
		return "", err
	}
	var resp NCCreateOrderResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("parse create order response failed: %w", err)
	}
	if resp.Code != 1 {
		return "", fmt.Errorf("create order failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if strings.TrimSpace(resp.PID) == "" {
		return "", fmt.Errorf("create order failed: empty pid")
	}
	return strings.TrimSpace(resp.PID), nil
}

func (c *NCClient) QueryOrderDetail(pid string) (*NCOrderDetail, error) {
	query := map[string]string{
		"type": "2",
		"pid":  strings.TrimSpace(pid),
	}
	raw, err := c.callAPI(http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}
	var resp NCQueryOrderResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse query order response failed: %w", err)
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("query order failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return &resp.Data, nil
}

func (c *NCClient) WaitOrderDone(pid string, waitMs int) (*NCOrderDetail, error) {
	if waitMs <= 0 {
		waitMs = ncDefaultWaitMs
	}
	deadline := time.Now().Add(time.Duration(waitMs) * time.Millisecond)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("poll timeout pid=%s", pid)
		}
		detail, err := c.QueryOrderDetail(pid)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(detail.Status) == "1" {
			return detail, nil
		}
		remain := time.Until(deadline)
		if remain <= 0 {
			return nil, fmt.Errorf("poll timeout pid=%s", pid)
		}
		sleep := time.Duration(ncPollIntervalMs) * time.Millisecond
		if sleep > remain {
			sleep = remain
		}
		time.Sleep(sleep)
	}
}

func (c *NCClient) GetOrderTextResult(queryType int, pid string) (string, error) {
	u, err := url.Parse(ncClientDownloadURL)
	if err != nil {
		return "", fmt.Errorf("parse download url failed: %w", err)
	}
	q := u.Query()
	q.Set("types", strconv.Itoa(queryType))
	q.Set("id", strings.TrimSpace(pid))
	q.Set("title", "undefined")
	q.Set("px", "id")
	q.Set("order", "desc")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create download request failed: %w", err)
	}
	cookieHeader := c.buildCookieHeader()
	if cookieHeader != "" {
		req.Header.Set("cookie", cookieHeader)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read download response failed: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("download status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}
	return string(body), nil
}

func parseNCClientQueryDR(raw string) []NCDRResult {
	rows := parseNCTextRows(raw)
	out := make([]NCDRResult, 0, len(rows))
	for _, row := range rows {
		item := NCDRResult{Columns: row.Columns, Raw: row.Raw}
		if len(row.Columns) > 0 {
			item.UIN = row.Columns[0]
		}
		if len(row.Columns) > 1 {
			str := strings.TrimSpace(row.Columns[1])
			str = strings.ReplaceAll(str, "天", "")
			item.PhoneOnlineDay, _ = strconv.Atoi(str)
			item.IsPhoneOnline = item.PhoneOnlineDay > 0
		}
		if len(row.Columns) > 2 {
			str := strings.TrimSpace(row.Columns[2])
			arr := strings.Split(str, "-")
			if len(arr) > 0 {
				item.IsPcOnline = arr[0] == "在线"
			}
			if len(arr) > 1 {
				item.Status = arr[1]
			}
		}
		out = append(out, item)
	}
	return out
}

func parseNCClientQueryQL(raw string) []NCQLResult {
	rows := parseNCTextRows(raw)
	out := make([]NCQLResult, 0, len(rows))
	for _, row := range rows {
		item := NCQLResult{Columns: row.Columns, Raw: row.Raw}
		if len(row.Columns) > 0 {
			item.UIN = row.Columns[0]
		}
		if len(row.Columns) > 1 {
			levelStr := strings.TrimSpace(row.Columns[1])
			levelStr = strings.ReplaceAll(levelStr, "级", "")
			item.Level, _ = strconv.Atoi(levelStr)
		}
		if len(row.Columns) > 2 {
			ageStr := strings.TrimSpace(row.Columns[2])
			ageStr = strings.ReplaceAll(ageStr, "年", "")
			item.Age, _ = strconv.Atoi(ageStr)
		}
		out = append(out, item)
	}
	return out
}

func (c *NCClient) callAPI(method string, query map[string]string, form url.Values) ([]byte, error) {
	u, err := url.Parse(ncClientAPIURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url failed: %w", err)
	}
	q := u.Query()
	q.Set("appid", c.AppID)
	q.Set("secret", c.Secret)
	for k, v := range query {
		if strings.TrimSpace(k) == "" {
			continue
		}
		q.Set(k, strings.TrimSpace(v))
	}
	u.RawQuery = q.Encode()

	var body io.Reader
	if method == http.MethodPost && form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create api request failed: %w", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read api response failed: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("api status=%s body=%s", resp.Status, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

func (c *NCClient) httpClient() *http.Client {
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 240 * time.Second}
	}
	return c.HTTPClient
}

func buildNCClientData(uinList []string) string {
	re := regexp.MustCompile(`\d+`)
	clean := make([]string, 0, len(uinList))
	for _, item := range uinList {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		num := re.FindString(item)
		if num == "" {
			continue
		}
		clean = append(clean, num)
	}
	return strings.Join(clean, "[sp]")
}

func parseNCTextRows(raw string) []NCRow {
	lines := strings.Split(raw, "\n")
	out := make([]NCRow, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\r", ""))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "注：") || strings.HasPrefix(line, "注:") {
			continue
		}
		if !strings.Contains(line, "----") {
			continue
		}
		parts := strings.Split(line, "----")
		row := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			row = append(row, p)
		}
		if len(row) > 0 {
			out = append(out, NCRow{Columns: row, Raw: line})
		}
	}
	return out
}

func (c *NCClient) buildCookieHeader() string {
	appID := strings.TrimSpace(c.AppID)
	ckMd5 := strings.TrimSpace(c.CKMd5)
	if appID == "" && ckMd5 == "" {
		return ""
	}
	if ckMd5 == "" {
		return "DedeUserID=" + appID
	}
	return "DedeUserID=" + appID + "; DedeUserID__ckMd5=" + ckMd5
}
