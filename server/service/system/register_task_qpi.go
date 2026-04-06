package system

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Material-Center/qpi"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"go.uber.org/zap"
)

const (
	captchaProviderYY   = "yy"
	captchaProviderAC   = "ac"
	captchaProviderFJ   = "fj"
	defaultShenlongArea = "210100,210200,210300,210400,210500,210600,210700,210800,210900,211000,211100,211200,211300,211400" // 辽宁省城市代码列表
	ip138GeoCachePrefix = "register:ip138:geo:"
)

var kuaidailiGetDPSURL = "https://dps.kdlapi.com/api/getdps"
var pingzanExtractURL = "https://service.ipzan.com/core-extract"

//go:embed pcc.json
var pccRawJSON []byte

var (
	pccInitOnce       sync.Once
	pccInitErr        error
	pccProvinceByName map[string]string   // 省名(归一化) -> 省code
	pccCityCodesByKey map[string][]string // 市名(归一化) -> 可能的市code
)

type shenlongGeo struct {
	Area   string `json:"area"`
	ISP    string `json:"isp"`
	Source string `json:"source"`
}

type captchaToken struct {
	Randstr string
	Ticket  string
}

func (s *RegisterTaskService) getCaptchaToken(cfg systemRegisterConfig, appID string, sid string) (*captchaToken, error) {
	global.GVA_LOG.Info("【注册任务】获取滑块验证码", zap.String("appID", appID), zap.String("sid", sid), zap.String("platform", cfg.CaptchaPlatform))
	provider := strings.ToLower(strings.TrimSpace(cfg.CaptchaPlatform))
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ { // 首次 + 失败后重试1次
		var token *captchaToken
		switch provider {
		case captchaProviderYY:
			token, lastErr = getCaptchaTokenFromYY(cfg, appID, sid)
		case captchaProviderAC:
			token, lastErr = getCaptchaTokenFromAC(cfg, appID, sid)
		case captchaProviderFJ:
			token, lastErr = getCaptchaTokenFromFJ(cfg, appID, sid)
		default:
			return nil, fmt.Errorf("不支持的验证码平台: %s", cfg.CaptchaPlatform)
		}
		if lastErr == nil {
			return token, nil
		}
	}
	return nil, fmt.Errorf("验证码获取失败(已重试1次): %w", lastErr)
}

func getCaptchaTokenFromYY(cfg systemRegisterConfig, appID string, sid string) (*captchaToken, error) {
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
		"sid":      strings.TrimSpace(sid),
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

func getCaptchaTokenFromAC(cfg systemRegisterConfig, appID string, sid string) (*captchaToken, error) {
	baseURL := strings.TrimSpace(cfg.CaptchaAccount)
	token := strings.TrimSpace(cfg.CaptchaToken)
	if baseURL == "" {
		baseURL = "http://39.99.146.154:16168"
	}
	if token == "" {
		return nil, errors.New("验证码平台AC token未配置")
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/captcha/run")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("token", token)
	q.Set("aid", appID)
	if strings.TrimSpace(sid) != "" {
		q.Set("sid", strings.TrimSpace(sid))
	}
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

func getCaptchaTokenFromFJ(cfg systemRegisterConfig, appID string, sid string) (*captchaToken, error) {
	baseURL := strings.TrimSpace(cfg.CaptchaAccount)
	token := strings.TrimSpace(cfg.CaptchaToken)
	if baseURL == "" {
		baseURL = "http://156.238.235.35:8860/"
	}
	if token == "" {
		return nil, errors.New("验证码平台FJ token未配置")
	}

	var buf bytes.Buffer
	buf.WriteString("token=" + url.QueryEscape(token))
	buf.WriteString("&newMethod=xkdth")
	buf.WriteString("&content=")
	content := "aid=" + strings.TrimSpace(appID) + "&sid=" + strings.TrimSpace(sid) + "&ip=&Url=" + "&uin="
	buf.WriteString(content)

	req, err := http.NewRequest(http.MethodPost, baseURL, strings.NewReader(buf.String()))
	if err != nil {
		return nil, fmt.Errorf("create fj request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fj request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read fj response failed: %w", err)
	}
	bodyText := strings.TrimSpace(string(body))
	ticket := substringBetween(bodyText, "ticket=", "&")
	randstr := substringBetween(bodyText, "randStr=", "&")
	if randstr == "" {
		randstr = substringBetween(bodyText, "randstr=", "&")
	}
	if ticket == "" || randstr == "" {
		return nil, fmt.Errorf("fj解析失败: %s", bodyText)
	}
	return &captchaToken{Randstr: randstr, Ticket: ticket}, nil
}

func (s *RegisterTaskService) allocateProxyURL(cfg systemRegisterConfig, phone string) (string, error) {
	geo := s.resolveProxyGeo(cfg, phone)
	global.GVA_LOG.Info("【注册任务】代理地区解析结果",
		zap.String("phone", strings.TrimSpace(phone)),
		zap.String("area", geo.Area),
		zap.String("isp", geo.ISP),
		zap.String("source", geo.Source),
	)
	return allocateProxyURLFromConfig(cfg, geo.Area, geo.ISP)
}

func allocateProxyURLFromConfig(cfg systemRegisterConfig, area string, isp string) (string, error) {
	if strings.TrimSpace(cfg.ProxyPlatform) == "" {
		return "", nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.ProxyPlatform)) {
	case "shenlong":
		key := strings.TrimSpace(cfg.ProxyAccount)
		sign := strings.TrimSpace(cfg.ProxyPassword)
		if key == "" || sign == "" {
			return "", errors.New("神龙代理配置不完整")
		}
		client := &http.Client{Timeout: 10 * time.Second}
		publicIP, err := fetchPublicIP(client)
		if err != nil {
			return "", err
		}
		if err := addShenlongWhitelist(client, sign, publicIP); err != nil {
			return "", err
		}
		addr, err := fetchShenlongSocks5(client, key, sign, area, isp)
		if err != nil {
			return "", err
		}
		return addr, nil
	case "kuaidaili":
		secretID := strings.TrimSpace(cfg.ProxySecretID)
		secretKey := strings.TrimSpace(cfg.ProxySecretKey)
		if secretID == "" || secretKey == "" {
			return "", errors.New("快代理配置不完整: SecretId/SecretKey 不能为空")
		}
		client := &http.Client{Timeout: 10 * time.Second}
		addr, err := fetchKuaidailiSocks5(client, secretID, secretKey, area, isp)
		if err != nil {
			return "", err
		}
		return addr, nil
	case "pingzan":
		no := strings.TrimSpace(cfg.ProxyAccount)
		secret := strings.TrimSpace(cfg.ProxyPassword)
		if no == "" || secret == "" {
			return "", errors.New("品赞代理配置不完整: no/secret 不能为空")
		}
		client := &http.Client{Timeout: 10 * time.Second}
		addr, err := fetchPingzanSocks5(client, no, secret, area)
		if err != nil {
			return "", err
		}
		return addr, nil
	default:
		return "", fmt.Errorf("不支持的代理平台: %s", cfg.ProxyPlatform)
	}
}

func (s *RegisterTaskService) resolveProxyGeo(cfg systemRegisterConfig, phone string) shenlongGeo {
	geo := shenlongGeo{
		Area:   strings.TrimSpace(defaultShenlongArea),
		ISP:    "",
		Source: "default",
	}
	phone = strings.TrimSpace(phone)
	if phone == "" || len(phone) != 11 {
		return geo
	}
	if cached, ok := getPhoneGeoFromCache(phone); ok {
		cached.Source = "cache"
		return cached
	}
	token := strings.TrimSpace(cfg.IP138Token)
	if token == "" {
		return geo
	}
	province, city, carrier, err := queryPhoneRegionFromIP138(token, phone)
	if err != nil {
		global.GVA_LOG.Warn(fmt.Sprintf("【注册任务】代理地区解析失败，回退默认地区 phone=%s err=%v", phone, err))
		return geo
	}
	if code, ok, resolveErr := resolveCityCodeFromPCC(province, city); resolveErr == nil && ok {
		geo.Area = code
	}
	geo.ISP = normalizeShenlongISP(carrier)
	geo.Source = "ip138"
	setPhoneGeoCache(phone, geo)
	return geo
}

func getPhoneGeoFromCache(phone string) (shenlongGeo, bool) {
	if global.GVA_REDIS == nil {
		return shenlongGeo{}, false
	}
	key := ip138GeoCachePrefix + strings.TrimSpace(phone)
	val, err := global.GVA_REDIS.Get(context.Background(), key).Result()
	if err != nil {
		return shenlongGeo{}, false
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return shenlongGeo{}, false
	}
	var geo shenlongGeo
	if err := json.Unmarshal([]byte(val), &geo); err != nil {
		return shenlongGeo{}, false
	}
	geo.Area = strings.TrimSpace(geo.Area)
	geo.ISP = normalizeShenlongISP(geo.ISP)
	if !isShenlongAreaValueValid(geo.Area) {
		return shenlongGeo{}, false
	}
	return geo, true
}

func setPhoneGeoCache(phone string, geo shenlongGeo) {
	if global.GVA_REDIS == nil {
		return
	}
	geo.Area = strings.TrimSpace(geo.Area)
	geo.ISP = normalizeShenlongISP(geo.ISP)
	geo.Source = ""
	if !isShenlongAreaValueValid(geo.Area) {
		return
	}
	raw, err := json.Marshal(geo)
	if err != nil {
		return
	}
	key := ip138GeoCachePrefix + strings.TrimSpace(phone)
	_ = global.GVA_REDIS.Set(context.Background(), key, string(raw), 24*time.Hour).Err()
}

func queryPhoneRegionFromIP138(token string, phone string) (province string, city string, carrier string, err error) {
	token = strings.TrimSpace(token)
	phone = strings.TrimSpace(phone)
	if token == "" || phone == "" {
		return "", "", "", errors.New("ip138 token或手机号为空")
	}
	u, err := url.Parse("https://api.ip138.com/mobile/")
	if err != nil {
		return "", "", "", err
	}
	q := u.Query()
	q.Set("mobile", phone)
	q.Set("datatype", "jsonp")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("token", token)
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", "", "", fmt.Errorf("ip138请求失败: %s", resp.Status)
	}
	payload := strings.TrimSpace(string(body))
	if payload == "" {
		return "", "", "", errors.New("ip138响应为空")
	}
	jsonBody := payload
	if !strings.HasPrefix(jsonBody, "{") {
		start := strings.Index(jsonBody, "{")
		end := strings.LastIndex(jsonBody, "}")
		if start < 0 || end <= start {
			return "", "", "", fmt.Errorf("ip138响应格式异常: %s", payload)
		}
		jsonBody = jsonBody[start : end+1]
	}
	var ret struct {
		Ret    string   `json:"ret"`
		Msg    string   `json:"msg"`
		Mobile string   `json:"mobile"`
		Data   []string `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonBody), &ret); err != nil {
		return "", "", "", err
	}
	if strings.ToLower(strings.TrimSpace(ret.Ret)) != "ok" {
		return "", "", "", fmt.Errorf("ip138返回失败: %s", strings.TrimSpace(ret.Msg))
	}
	if len(ret.Data) < 3 {
		return "", "", "", fmt.Errorf("ip138数据异常: %s", jsonBody)
	}
	province = strings.TrimSpace(ret.Data[0])
	city = strings.TrimSpace(ret.Data[1])
	carrier = strings.TrimSpace(ret.Data[2])
	return province, city, carrier, nil
}

func resolveCityCodeFromPCC(province, city string) (string, bool, error) {
	if err := initPCCIndexes(); err != nil {
		return "", false, err
	}
	cityKey := normalizeRegionName(city)
	if cityKey == "" {
		return "", false, nil
	}
	cityCandidates := pccCityCodesByKey[cityKey]
	if len(cityCandidates) == 0 {
		return "", false, nil
	}

	provinceKey := normalizeRegionName(province)
	if provinceKey != "" {
		if provinceCode, ok := pccProvinceByName[provinceKey]; ok && len(provinceCode) >= 2 {
			prefix := provinceCode[:2]
			for _, code := range cityCandidates {
				if strings.HasPrefix(code, prefix) {
					return code, true, nil
				}
			}
		}
	}
	return cityCandidates[0], true, nil
}

func initPCCIndexes() error {
	pccInitOnce.Do(func() {
		var kv map[string]string
		if err := json.Unmarshal(pccRawJSON, &kv); err != nil {
			pccInitErr = fmt.Errorf("解析pcc.json失败: %w", err)
			return
		}
		pccProvinceByName = map[string]string{}
		pccCityCodesByKey = map[string][]string{}
		for code, name := range kv {
			code = strings.TrimSpace(code)
			name = strings.TrimSpace(name)
			if !isStrictCityCode(code) || name == "" {
				continue
			}
			if isProvinceLevelCode(code) {
				pccProvinceByName[normalizeRegionName(name)] = code
				continue
			}
			if isCityLevelCode(code) {
				key := normalizeRegionName(name)
				if key == "" {
					continue
				}
				pccCityCodesByKey[key] = append(pccCityCodesByKey[key], code)
			}
		}
	})
	return pccInitErr
}

func normalizeRegionName(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, " ", ""))
	if name == "" {
		return ""
	}
	for _, suffix := range []string{
		"维吾尔自治区", "壮族自治区", "回族自治区", "自治区", "特别行政区",
		"自治州", "地区", "盟", "省", "市", "州",
	} {
		name = strings.TrimSuffix(name, suffix)
	}
	return strings.TrimSpace(name)
}

func normalizeShenlongISP(raw string) string {
	s := strings.TrimSpace(raw)
	switch {
	case strings.Contains(s, "联通"):
		return "联通"
	case strings.Contains(s, "电信"):
		return "电信"
	case strings.Contains(s, "移动"):
		return "移动"
	default:
		return ""
	}
}

func isStrictCityCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isProvinceLevelCode(code string) bool {
	return isStrictCityCode(code) && strings.HasSuffix(code, "0000")
}

func isCityLevelCode(code string) bool {
	return isStrictCityCode(code) && strings.HasSuffix(code, "00") && !strings.HasSuffix(code, "0000")
}

func isShenlongAreaValueValid(area string) bool {
	area = strings.TrimSpace(area)
	if area == "" {
		return false
	}
	for _, part := range strings.Split(area, ",") {
		part = strings.TrimSpace(part)
		if part == "" || !isStrictCityCode(part) {
			return false
		}
	}
	return true
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

func fetchShenlongSocks5(client *http.Client, key, sign, area, isp string) (string, error) {
	u, _ := url.Parse("http://api.shenlongip.com/ip")
	q := u.Query()
	q.Set("key", key)
	q.Set("sign", sign)
	q.Set("count", "1")
	q.Set("pattern", "json")
	q.Set("mr", "1")
	q.Set("protocol", "3")
	q.Set("type", "3")
	if strings.TrimSpace(area) != "" {
		q.Set("area", strings.TrimSpace(area))
	}
	ispRet := normalizeShenlongISP(isp)
	if ispRet != "" {
		q.Set("isp", ispRet)
	}
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

func fetchKuaidailiSocks5(client *http.Client, secretID, secretKey, area, isp string) (string, error) {
	method := "GET"
	parsed, err := url.Parse(kuaidailiGetDPSURL)
	if err != nil {
		return "", err
	}
	params := map[string]string{
		"secret_id": secretID,
		"sign_type": "hmacsha1",
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		"num":       "1",
		"format":    "text",
	}
	if strings.TrimSpace(area) != "" {
		params["area"] = strings.TrimSpace(area)
	}
	if carrier := normalizeKuaidailiCarrier(isp); carrier != "" {
		params["carrier"] = carrier
	}
	params["signature"] = signKuaidailiRequest(method, parsed.Path, params, secretKey)

	q := parsed.Query()
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q.Set(k, params[k])
	}
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("快代理请求失败: %s", resp.Status)
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "", errors.New("快代理返回为空")
	}
	if strings.HasPrefix(text, "ERROR(") {
		return "", fmt.Errorf("快代理返回失败: %s", text)
	}
	line := strings.TrimSpace(strings.Split(text, "\n")[0])
	host, port, splitErr := net.SplitHostPort(line)
	if splitErr != nil {
		return "", fmt.Errorf("快代理返回格式异常: %s", line)
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
		return "", fmt.Errorf("快代理返回ip端口为空: %s", line)
	}
	return "socks5://" + net.JoinHostPort(host, port), nil
}

func fetchPingzanSocks5(client *http.Client, no, secret, area string) (string, error) {
	query := url.Values{
		"no":       []string{no},
		"secret":   []string{secret},
		"num":      []string{"1"},
		"mode":     []string{"auth"},
		"minute":   []string{"5"},
		"pool":     []string{"quality"},
		"format":   []string{"json"},
		"protocol": []string{"3"},
	}
	if resolvedArea := normalizePingzanArea(area); resolvedArea != "" {
		query.Set("area", resolvedArea)
	}
	return fetchPingzanSocks5WithQuery(client, query)
}

func fetchPingzanSocks5WithQuery(client *http.Client, query url.Values) (string, error) {
	u, err := url.Parse(pingzanExtractURL)
	if err != nil {
		return "", err
	}
	u.RawQuery = query.Encode()
	req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("品赞代理请求失败: %s", resp.Status)
	}
	var ret struct {
		Code    int    `json:"code"`
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    struct {
			List []struct {
				IP       string          `json:"ip"`
				Port     json.RawMessage `json:"port"`
				Account  string          `json:"account"`
				Password string          `json:"password"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &ret); err != nil {
		return "", fmt.Errorf("解析品赞响应失败: %w", err)
	}
	if ret.Code != 0 {
		msg := strings.TrimSpace(ret.Message)
		if msg == "" {
			msg = "未知错误"
		}
		return "", fmt.Errorf("品赞代理提取失败: %s", msg)
	}
	if len(ret.Data.List) == 0 {
		return "", errors.New("品赞代理返回为空")
	}
	first := ret.Data.List[0]
	host := strings.TrimSpace(first.IP)
	port, err := parsePingzanPort(first.Port)
	if err != nil || host == "" {
		return "", errors.New("品赞代理返回IP或端口为空")
	}
	if strings.TrimSpace(first.Account) != "" {
		return (&url.URL{
			Scheme: "socks5",
			User:   url.UserPassword(strings.TrimSpace(first.Account), strings.TrimSpace(first.Password)),
			Host:   net.JoinHostPort(host, port),
		}).String(), nil
	}
	return "socks5://" + net.JoinHostPort(host, port), nil
}

func parsePingzanPort(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("empty port")
	}
	var portInt int
	if err := json.Unmarshal(raw, &portInt); err == nil {
		if portInt <= 0 {
			return "", errors.New("invalid port")
		}
		return strconv.Itoa(portInt), nil
	}
	var portStr string
	if err := json.Unmarshal(raw, &portStr); err != nil {
		return "", err
	}
	portStr = strings.TrimSpace(portStr)
	if portStr == "" {
		return "", errors.New("invalid port")
	}
	portInt, err := strconv.Atoi(portStr)
	if err != nil || portInt <= 0 {
		return "", errors.New("invalid port")
	}
	return portStr, nil
}

func normalizePingzanArea(area string) string {
	area = strings.TrimSpace(area)
	if area == "" {
		return ""
	}
	for _, part := range strings.Split(area, ",") {
		part = strings.TrimSpace(part)
		if isStrictCityCode(part) {
			return part
		}
	}
	if isStrictCityCode(area) {
		return area
	}
	return ""
}

func signKuaidailiRequest(method, path string, params map[string]string, secretKey string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	queryPairs := make([]string, 0, len(keys))
	for _, k := range keys {
		queryPairs = append(queryPairs, k+"="+params[k])
	}
	raw := method + path + "?" + strings.Join(queryPairs, "&")
	mac := hmac.New(sha1.New, []byte(secretKey))
	_, _ = mac.Write([]byte(raw))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func normalizeKuaidailiCarrier(raw string) string {
	switch normalizeShenlongISP(raw) {
	case "联通":
		return "1"
	case "电信":
		return "2"
	case "移动":
		return "3"
	default:
		return ""
	}
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

func buildTLV544ProviderFromConfig(cfg systemRegisterConfig) (qpi.TLV544Provider, error) {
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.ApiBase), "/")
	apiToken := strings.TrimSpace(cfg.ApiToken)
	if apiBase == "" || apiToken == "" {
		return nil, errors.New("登录签名服务 apiBase/apiToken 未配置")
	}
	return func(req qpi.TLV544Request) ([]byte, error) {
		retries := 3
		for attempts := 0; ; attempts++ {
			form := url.Values{}
			form.Set("token", apiToken)
			form.Set("uin", strings.TrimSpace(req.UIN))
			form.Set("salt", strings.TrimSpace(req.Salt))
			form.Set("data", strings.ReplaceAll(strings.TrimSpace(req.SaltData), " ", ""))

			respText, err := doPostForm(apiBase+"/energy", form.Encode())
			if err != nil {
				if attempts < retries {
					continue
				}
				return nil, fmt.Errorf("makeTLV544 energy request failed: %w", err)
			}
			respText = strings.TrimSpace(respText)
			if respText == "" {
				return nil, errors.New("makeTLV544 energy response is empty")
			}
			return hex.DecodeString(strings.ReplaceAll(respText, " ", ""))
		}
	}, nil
}

func buildTLV553ProviderFromConfig(cfg systemRegisterConfig) (func(uin uint32) ([]byte, error), error) {
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.ApiBase), "/")
	if apiBase == "" {
		return nil, errors.New("登录签名服务 apiBase 未配置")
	}
	return func(uin uint32) ([]byte, error) {
		endpoint := apiBase + "/get_xw_debug_id"
		bodyRaw := "uin=" + strconv.FormatUint(uint64(uin), 10)
		respText, err := doPostForm(endpoint, bodyRaw)
		if err != nil {
			return nil, fmt.Errorf("tlv553 request failed: %w", err)
		}
		var payload struct {
			Code int    `json:"code"`
			Data string `json:"data"`
		}
		if err := json.Unmarshal([]byte(respText), &payload); err != nil {
			return nil, fmt.Errorf("parse tlv553 failed: %w, body=%s", err, respText)
		}
		if payload.Code != 0 {
			return nil, fmt.Errorf("tlv553 failed: code=%d body=%s", payload.Code, respText)
		}
		return hex.DecodeString(strings.ReplaceAll(payload.Data, " ", ""))
	}, nil
}

func buildSignProviderFromConfig(cfg systemRegisterConfig) (qpi.SignProvider, error) {
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.ApiBase), "/")
	apiToken := strings.TrimSpace(cfg.ApiToken)
	if apiBase == "" || apiToken == "" {
		return nil, errors.New("登录签名服务 apiBase/apiToken 未配置")
	}
	return func(req qpi.SignRequest) ([]byte, error) {
		form := url.Values{}
		form.Set("token", apiToken)
		form.Set("uin", strconv.FormatUint(uint64(req.UIN), 10))
		form.Set("cmd", req.Cmd)
		form.Set("data", strings.ToLower(hex.EncodeToString(req.Body)))
		form.Set("qua", strings.TrimSpace(req.QUA))
		form.Set("seq", strconv.Itoa(req.Seq))
		form.Set("guid", strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(req.GUIDHex)), " ", ""))
		form.Set("android_id", strings.TrimSpace(req.AndroidID))
		form.Set("qimei36", strings.TrimSpace(req.QIMEI36))
		respText, err := doPostForm(apiBase+"/qsign", form.Encode())
		if err != nil {
			return nil, fmt.Errorf("sign provider request failed: %w", err)
		}
		return hex.DecodeString(strings.ReplaceAll(strings.TrimSpace(respText), " ", ""))
	}, nil
}

func buildInitProviderFromConfig(cfg systemRegisterConfig) (qpi.InitProvider, error) {
	apiBase := strings.TrimRight(strings.TrimSpace(cfg.ApiBase), "/")
	apiToken := strings.TrimSpace(cfg.ApiToken)
	if apiBase == "" || apiToken == "" {
		return nil, errors.New("登录签名服务 apiBase/apiToken 未配置")
	}
	return func(req qpi.InitRequest) error {
		form := url.Values{}
		form.Set("token", apiToken)
		form.Set("uin", req.UIN)
		form.Set("ssoseq", strconv.Itoa(req.SSOSeq))
		form.Set("appid", strconv.Itoa(req.AppID))
		form.Set("appids", strconv.Itoa(req.AppID2))
		form.Set("qqver", req.QQVer)
		form.Set("guid", req.GUIDHex)
		form.Set("qimei36", req.QIMEI36)
		form.Set("qua", req.QUA)
		form.Set("apkver", req.APKVer)
		form.Set("code", strconv.Itoa(req.Code))
		form.Set("android_id", req.AndroidID)
		form.Set("brand", req.Brand)
		form.Set("model", req.Model)
		form.Set("osver", req.OSVer)
		form.Set("cookie", req.CookieHex)
		rsp, err := doPostForm(apiBase+"/init", form.Encode())
		if err != nil {
			return fmt.Errorf("init provider request failed: %w", err)
		}
		if strings.TrimSpace(rsp) != "error" {
			return nil
		}
		return fmt.Errorf("init provider response failed: %s", rsp)
	}, nil
}

func doPostForm(targetURL string, bodyRaw string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBufferString(bodyRaw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("doPOST request failed: %s", resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func substringBetween(s, left, right string) string {
	start := strings.Index(s, left)
	if start < 0 {
		return ""
	}
	start += len(left)
	remain := s[start:]
	end := strings.Index(remain, right)
	if end < 0 {
		return strings.TrimSpace(remain)
	}
	return strings.TrimSpace(remain[:end])
}

type systemRegisterConfig struct {
	DefaultPassword string
	NaichaAppID     string
	NaichaSecret    string
	NaichaCKMd5     string
	IP138Token      string
	ApiBase         string
	ApiToken        string
	ProxyPlatform   string
	ProxyAccount    string
	ProxyPassword   string
	ProxySecretID   string
	ProxySecretKey  string
	CaptchaPlatform string
	CaptchaAccount  string
	CaptchaPassword string
	CaptchaToken    string
}

func (s *RegisterTaskService) getRegisterRuntimeConfig(leaderID *uint) (systemRegisterConfig, error) {
	cfg := systemRegisterConfig{}
	_ = leaderID
	var adminCfg struct {
		DefaultPassword string
		NaichaAppID     string
		NaichaSecret    string
		NaichaCKMd5     string
		IP138Token      string
		ApiBase         string
		ApiToken        string
		ProxyPlatform   string
		ProxyAccount    string
		ProxyPassword   string
		ProxySecretID   string
		ProxySecretKey  string
		CaptchaPlatform string
		CaptchaAccount  string
		CaptchaPassword string
		CaptchaToken    string
	}
	if err := global.GVA_DB.Model(&system.SysRegisterConfig{}).
		Select("default_password, naicha_app_id, naicha_secret, naicha_ck_md5, ip138_token, api_base, api_token, proxy_platform, proxy_account, proxy_password, proxy_secret_id, proxy_secret_key, captcha_platform, captcha_account, captcha_password, captcha_token").
		Where("owner_type = ? AND owner_id = 0", system.RegisterConfigOwnerAdmin).
		First(&adminCfg).Error; err == nil {
		cfg.DefaultPassword = adminCfg.DefaultPassword
		cfg.NaichaAppID = adminCfg.NaichaAppID
		cfg.NaichaSecret = adminCfg.NaichaSecret
		cfg.NaichaCKMd5 = adminCfg.NaichaCKMd5
		cfg.IP138Token = adminCfg.IP138Token
		cfg.ApiBase = adminCfg.ApiBase
		cfg.ApiToken = adminCfg.ApiToken
		cfg.ProxyPlatform = adminCfg.ProxyPlatform
		cfg.ProxyAccount = adminCfg.ProxyAccount
		cfg.ProxyPassword = adminCfg.ProxyPassword
		cfg.ProxySecretID = adminCfg.ProxySecretID
		cfg.ProxySecretKey = adminCfg.ProxySecretKey
		cfg.CaptchaPlatform = adminCfg.CaptchaPlatform
		cfg.CaptchaAccount = adminCfg.CaptchaAccount
		cfg.CaptchaPassword = adminCfg.CaptchaPassword
		cfg.CaptchaToken = adminCfg.CaptchaToken
	}
	return cfg, nil
}
