package system

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

const (
	captchaProviderYY = "yy"
	captchaProviderAC = "ac"
)

type captchaToken struct {
	Randstr string
	Ticket  string
}

func (s *RegisterTaskService) getCaptchaToken(cfg systemRegisterConfig, appID string) (*captchaToken, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.CaptchaPlatform))
	switch provider {
	case captchaProviderYY:
		return getCaptchaTokenFromYY(cfg, appID)
	case captchaProviderAC:
		return getCaptchaTokenFromAC(cfg, appID)
	default:
		return nil, fmt.Errorf("不支持的验证码平台: %s", cfg.CaptchaPlatform)
	}
}

func getCaptchaTokenFromYY(cfg systemRegisterConfig, appID string) (*captchaToken, error) {
	username := strings.TrimSpace(cfg.CaptchaAccount)
	password := strings.TrimSpace(cfg.CaptchaPassword)
	if username == "" || password == "" {
		return nil, errors.New("验证码平台YY账号或密码未配置")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	submitURL := "http://yy.svip168.vip/api/v1/submit"
	queryURL := "http://yy.svip168.vip/api/v1/query"

	payload, _ := json.Marshal(map[string]any{
		"username": username,
		"password": password,
		"aid":      appID,
	})
	req, err := http.NewRequest(http.MethodPost, submitURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var submit struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &submit); err != nil {
		return nil, fmt.Errorf("解析YY提交响应失败: %w", err)
	}
	if submit.Code != 200 {
		return nil, fmt.Errorf("YY提交失败: %s", submit.Msg)
	}
	orderID := strings.TrimSpace(submit.Data)
	if orderID == "" {
		return nil, errors.New("YY返回订单为空")
	}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		queryReq, _ := http.NewRequest(http.MethodGet, queryURL+"?order="+url.QueryEscape(orderID), nil)
		queryResp, err := client.Do(queryReq)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		body, _ := io.ReadAll(queryResp.Body)
		queryResp.Body.Close()

		var query struct {
			Code    int    `json:"code"`
			Msg     string `json:"msg"`
			Data    string `json:"data"`
			Randstr string `json:"randstr"`
			Ticket  string `json:"ticket"`
		}
		if err := json.Unmarshal(body, &query); err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		if query.Code == 100 {
			time.Sleep(3 * time.Second)
			continue
		}
		if query.Code != 200 {
			return nil, fmt.Errorf("YY查询失败: %s", query.Msg)
		}

		if query.Randstr != "" && query.Ticket != "" {
			return &captchaToken{Randstr: query.Randstr, Ticket: query.Ticket}, nil
		}
		var dataObj map[string]any
		if err := json.Unmarshal([]byte(query.Data), &dataObj); err == nil {
			randstr, _ := dataObj["randstr"].(string)
			ticket, _ := dataObj["ticket"].(string)
			if randstr != "" && ticket != "" {
				return &captchaToken{Randstr: randstr, Ticket: ticket}, nil
			}
		}
		return nil, errors.New("YY未返回有效randstr/ticket")
	}
	return nil, errors.New("YY等待滑块超时")
}

func getCaptchaTokenFromAC(cfg systemRegisterConfig, appID string) (*captchaToken, error) {
	baseURL := strings.TrimSpace(cfg.CaptchaAccount)
	token := strings.TrimSpace(cfg.CaptchaToken)
	if baseURL == "" {
		baseURL = "http://39.99.146.154:16168"
	}
	if token == "" {
		return nil, errors.New("验证码平台AC token未配置")
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/tencent/captcha/run")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("token", token)
	q.Set("aid", appID)
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 50 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var acResp struct {
		ErrorCode string `json:"errorCode"`
		Randstr   string `json:"randstr"`
		Ticket    string `json:"ticket"`
	}
	if err := json.Unmarshal(body, &acResp); err != nil {
		return nil, fmt.Errorf("解析AC响应失败: %w", err)
	}
	if acResp.ErrorCode != "0" {
		return nil, fmt.Errorf("AC返回失败: %s", acResp.ErrorCode)
	}
	if acResp.Randstr == "" || acResp.Ticket == "" {
		return nil, errors.New("AC未返回有效randstr/ticket")
	}
	return &captchaToken{Randstr: acResp.Randstr, Ticket: acResp.Ticket}, nil
}

func (s *RegisterTaskService) allocateProxyURL(cfg systemRegisterConfig) (string, error) {
	return allocateProxyURLFromConfig(cfg)
}

func allocateProxyURLFromConfig(cfg systemRegisterConfig) (string, error) {
	if strings.TrimSpace(cfg.ProxyPlatform) == "" {
		return "", nil
	}
	if strings.ToLower(strings.TrimSpace(cfg.ProxyPlatform)) != "shenlong" {
		return "", fmt.Errorf("不支持的代理平台: %s", cfg.ProxyPlatform)
	}

	key := strings.TrimSpace(cfg.ProxyAccount)
	sign := strings.TrimSpace(cfg.ProxyPassword)
	if key == "" || sign == "" {
		return "", errors.New("代理平台配置不完整")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	publicIP, err := fetchPublicIP(client)
	if err != nil {
		return "", err
	}
	if err := addShenlongWhitelist(client, sign, publicIP); err != nil {
		return "", err
	}
	addr, err := fetchShenlongSocks5(client, key, sign)
	if err != nil {
		return "", err
	}
	return addr, nil
}

func fetchPublicIP(client *http.Client) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://ip.sb?format=text", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", errors.New("获取出口IP失败")
	}
	if _, err := netip.ParseAddr(ip); err != nil {
		return "", fmt.Errorf("无效出口IP: %w", err)
	}
	return ip, nil
}

func addShenlongWhitelist(client *http.Client, sign string, ip string) error {
	u, _ := url.Parse("http://api.shenlongip.com/white/add")
	q := u.Query()
	q.Set("key", "ayay123")
	q.Set("sign", sign)
	q.Set("ip", ip)
	u.RawQuery = q.Encode()
	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var ret struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &ret); err != nil {
		return err
	}
	if ret.Code != 200 && ret.Code != 1007 {
		return fmt.Errorf("神龙白名单失败: %s", ret.Msg)
	}
	return nil
}

func fetchShenlongSocks5(client *http.Client, key, sign string) (string, error) {
	u, _ := url.Parse("http://api.shenlongip.com/ip")
	q := u.Query()
	q.Set("key", key)
	q.Set("sign", sign)
	q.Set("count", "1")
	q.Set("pattern", "json")
	q.Set("mr", "1")
	q.Set("protocol", "3")
	q.Set("type", "3")
	u.RawQuery = q.Encode()

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var ret struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data []struct {
				IP   string `json:"ip"`
				Port int    `json:"port"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &ret); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if ret.Code == 200 && len(ret.Data) > 0 {
			host := net.JoinHostPort(ret.Data[0].IP, strconv.Itoa(ret.Data[0].Port))
			return "socks5://" + host, nil
		}
		time.Sleep(2 * time.Second)
	}
	return "", errors.New("神龙代理提取失败")
}

func buildCacheINI(cache map[string]string) string {
	if len(cache) == 0 {
		return ""
	}
	sb := strings.Builder{}
	uin := strings.TrimSpace(cache["uin"])
	sb.WriteString("[")
	sb.WriteString(uin)
	sb.WriteString("]\n")
	for k, v := range cache {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	return sb.String()
}

type systemRegisterConfig struct {
	DefaultPassword string
	NaichaAppID     string
	NaichaSecret    string
	NaichaCKMd5     string
	ProxyPlatform   string
	ProxyAccount    string
	ProxyPassword   string
	CaptchaPlatform string
	CaptchaAccount  string
	CaptchaPassword string
	CaptchaToken    string
}

func (s *RegisterTaskService) getRegisterRuntimeConfig(leaderID *uint) (systemRegisterConfig, error) {
	cfg := systemRegisterConfig{}
	if leaderID != nil && *leaderID != 0 {
		var leaderCfg struct {
			ProxyPlatform   string
			ProxyAccount    string
			ProxyPassword   string
			CaptchaPlatform string
			CaptchaAccount  string
			CaptchaPassword string
			CaptchaToken    string
		}
		err := global.GVA_DB.Model(&system.SysRegisterConfig{}).
			Select("proxy_platform, proxy_account, proxy_password, captcha_platform, captcha_account, captcha_password, captcha_token").
			Where("owner_type = ? AND owner_id = ?", system.RegisterConfigOwnerLeader, *leaderID).
			First(&leaderCfg).Error
		if err == nil {
			cfg.ProxyPlatform = leaderCfg.ProxyPlatform
			cfg.ProxyAccount = leaderCfg.ProxyAccount
			cfg.ProxyPassword = leaderCfg.ProxyPassword
			cfg.CaptchaPlatform = leaderCfg.CaptchaPlatform
			cfg.CaptchaAccount = leaderCfg.CaptchaAccount
			cfg.CaptchaPassword = leaderCfg.CaptchaPassword
			cfg.CaptchaToken = leaderCfg.CaptchaToken
		}
	}
	var adminCfg struct {
		DefaultPassword string
		NaichaAppID     string
		NaichaSecret    string
		NaichaCKMd5     string
	}
	if err := global.GVA_DB.Model(&system.SysRegisterConfig{}).
		Select("default_password, naicha_app_id, naicha_secret, naicha_ck_md5").
		Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).
		First(&adminCfg).Error; err == nil {
		cfg.DefaultPassword = adminCfg.DefaultPassword
		cfg.NaichaAppID = adminCfg.NaichaAppID
		cfg.NaichaSecret = adminCfg.NaichaSecret
		cfg.NaichaCKMd5 = adminCfg.NaichaCKMd5
	}
	return cfg, nil
}
