package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ServerConfig struct {
	Crypto CryptoConfig
	Now    func() time.Time
	Random io.Reader
	Store  *Store

	LogOutput io.Writer
	EnvProxy  *EnvProxyConfig
}

type EnvProxyConfig struct {
	BaseURL    string
	Path       string
	Key        string
	HTTPClient *http.Client
}

type Server struct {
	cfg   ServerConfig
	logMu sync.Mutex
}

func NewServer(cfg ServerConfig) *Server {
	if cfg.Crypto.Seed == "" {
		cfg.Crypto = DefaultConfig()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Random == nil {
		cfg.Random = rand.Reader
	}
	if cfg.LogOutput == nil {
		cfg.LogOutput = os.Stdout
	}
	return &Server{cfg: cfg}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/shanghaitime", s.handleShanghaiTime)
	mux.HandleFunc("/get_device", s.handleGetDevice)
	mux.HandleFunc("/use_code", s.handleUseCode)
	mux.HandleFunc("/stoptime", s.handleStopTime)
	mux.HandleFunc("/上传", s.handleUpload)
	return s.accessLog("all", mux)
}

func (s *Server) AuthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/shanghaitime", s.handleShanghaiTime)
	mux.HandleFunc("/get_device", s.handleGetDevice)
	mux.HandleFunc("/use_code", s.handleUseCode)
	mux.HandleFunc("/stoptime", s.handleStopTime)
	return s.accessLog("auth", mux)
}

func (s *Server) UploadHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/上传", s.handleUpload)
	return s.accessLog("upload", mux)
}

func (s *Server) EnvHandler() http.Handler {
	mux := http.NewServeMux()
	if s.cfg.EnvProxy != nil {
		mux.HandleFunc("/", s.handleEnvProxy)
		return s.accessLog("env", mux)
	}
	mux.HandleFunc("/add_env", s.handleAddEnv)
	mux.HandleFunc("/get_env", s.handleGetEnv)
	mux.HandleFunc("/query_env_list", s.handleQueryEnvList)
	mux.HandleFunc("/query_env", s.handleQueryEnv)
	mux.HandleFunc("/freeze_env", s.handleFreezeEnv)
	mux.HandleFunc("/unfreeze_env", s.handleUnfreezeEnv)
	mux.HandleFunc("/freeze_by_condition", s.handleFreezeByCondition)
	mux.HandleFunc("/unfreeze_by_condition", s.handleUnfreezeByCondition)
	mux.HandleFunc("/delete_env", s.handleDeleteEnv)
	mux.HandleFunc("/delete_by_condition", s.handleDeleteByCondition)
	mux.HandleFunc("/clean_env", s.handleCleanEnv)
	mux.HandleFunc("/query_by_device", s.handleQueryByDevice)
	mux.HandleFunc("/get_env_enhanced", s.handleGetEnvEnhanced)
	mux.HandleFunc("/get_env_enhanced2", s.handleGetEnvEnhanced2)
	mux.HandleFunc("/get_env_for_make", s.handleGetEnvForMake)
	mux.HandleFunc("/make_success", s.handleMakeSuccess)
	mux.HandleFunc("/increase_make_count", s.handleIncreaseMakeCount)
	mux.HandleFunc("/decrease_make_count", s.handleDecreaseMakeCount)
	mux.HandleFunc("/reset_make_count", s.handleResetMakeCount)
	mux.HandleFunc("/stats_by_type", s.handleStatsByType)
	mux.HandleFunc("/stats_make_progress", s.handleStatsMakeProgress)
	mux.HandleFunc("/total", s.handleTotal)
	mux.HandleFunc("/available", s.handleAvailable)
	mux.HandleFunc("/frozen", s.handleFrozen)
	mux.HandleFunc("/unused", s.handleUnused)
	mux.HandleFunc("/get_env_fixed", s.handleGetEnvFixed)
	mux.HandleFunc("/stats", s.handleEnvStats)
	return s.accessLog("env", mux)
}

func (s *Server) handleEnvProxy(w http.ResponseWriter, r *http.Request) {
	proxy := s.cfg.EnvProxy
	if proxy == nil || strings.TrimSpace(proxy.BaseURL) == "" || strings.TrimSpace(proxy.Key) == "" {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("environment proxy is not configured"))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("read environment request: %w", err))
		return
	}
	path := proxy.Path
	if path == "" {
		path = "/internalTool/miEnv"
	}
	targetPath := r.URL.EscapedPath()
	if targetPath == "" {
		targetPath = "/"
	}
	target := strings.TrimRight(proxy.BaseURL, "/") + "/" + strings.Trim(path, "/") + targetPath
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	request, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("build environment proxy request: %w", err))
		return
	}
	copyRequestHeaders(request.Header, r.Header)
	request.Header.Set("X-MI-Internal-Key", proxy.Key)
	// Avoid transparent gzip negotiation: the compiled client expects to parse
	// the JSON envelope directly, and identity makes byte counts/logs reliable.
	request.Header.Set("Accept-Encoding", "identity")
	client := proxy.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	upstreamStarted := time.Now()
	response, err := client.Do(request)
	if err != nil {
		s.logf("proxy service=env method=%s path=%q upstream=%q error=%q duration=%s", r.Method, r.URL.RequestURI(), target, err, time.Since(upstreamStarted).Round(time.Microsecond))
		writeError(w, http.StatusBadGateway, fmt.Errorf("environment proxy request failed: %w", err))
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		if isHopByHopHeader(key) || strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if response.ContentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(response.ContentLength, 10))
	}
	w.WriteHeader(response.StatusCode)
	bytesWritten, copyErr := io.Copy(w, response.Body)
	copyError := "-"
	if copyErr != nil {
		copyError = copyErr.Error()
	}
	s.logf("proxy service=env method=%s path=%q upstream_status=%d upstream_bytes=%d duration=%s error=%q", r.Method, r.URL.RequestURI(), response.StatusCode, bytesWritten, time.Since(upstreamStarted).Round(time.Microsecond), copyError)
}

func (s *Server) accessLog(service string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(lw, r)

		s.logf("request service=%s method=%s path=%q status=%d bytes=%d duration=%s remote=%q content_length=%d user_agent=%q", service, r.Method, r.URL.RequestURI(), lw.status, lw.bytes, time.Since(start).Round(time.Microsecond), r.RemoteAddr, r.ContentLength, r.UserAgent())
	})
}

func (s *Server) logf(format string, args ...any) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	fmt.Fprintf(s.cfg.LogOutput, "time=%s "+format+"\n", append([]any{time.Now().Format(time.RFC3339Nano)}, args...)...)
}

func copyRequestHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) || strings.EqualFold(key, "Host") || strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.ResponseWriter.WriteHeader(http.StatusOK)
		w.wroteHeader = true
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (w *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *loggingResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		if !w.wroteHeader {
			w.ResponseWriter.WriteHeader(http.StatusOK)
			w.wroteHeader = true
		}
		f.Flush()
	}
}

func (s *Server) handleShanghaiTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
		return
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	plain := s.cfg.Now().In(loc).Format("2006-01-02 15:04:05")
	encrypted, err := encryptResponseStringAt(plain, s.cfg.Crypto, s.cfg.Now(), s.cfg.Random)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code": 200,
		"data": encrypted,
	})
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
	}
	if r.Method == http.MethodGet {
		req.DeviceID = r.URL.Query().Get("device_id")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
			return
		}
	}
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("device_id is required"))
		return
	}

	now := s.cfg.Now().In(shanghaiLocation())
	// Local compatibility mode is intentionally stateless: every device gets
	// a long-lived synthetic authorization response. The local SQLite license
	// table remains available for older databases but is not consulted here.
	startedAt := now
	authorizedUntil := now.AddDate(100, 0, 0)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"设备id":    req.DeviceID,
		"开始时间":    startedAt.Format("2006-01-02 15:04"),
		"到期时间":    authorizedUntil.Format("2006-01-02 15:04:05"),
		"天数":      int(authorizedUntil.Sub(now).Hours() / 24),
	})
}

func (s *Server) handleUseCode(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
		return
	}
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("device_id is required"))
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("code is required"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": false,
		"error":   "失败,授权码无效",
	})
}

func (s *Server) handleStopTime(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}

	var req struct {
		EncryptedDevice string `json:"encrypted_device"`
		EncryptedKey    string `json:"encrypted_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
		return
	}
	if req.EncryptedDevice == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("encrypted_device is required"))
		return
	}
	deviceID, err := decryptString(req.EncryptedDevice, s.cfg.Crypto)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decrypt encrypted_device: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"stopped": false,
		"设备id":    deviceID,
	})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
		return
	}

	for _, field := range []string{"设备", "当前时间", "手机号", "账号", "密码"} {
		value := req[field]
		if value == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%s is required", field))
			return
		}
		plain, err := decryptString(value, s.cfg.Crypto)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("decrypt %s: %w", field, err))
			return
		}
		_ = plain
	}
	if s.cfg.Store != nil {
		_, err := s.cfg.Store.SaveUpload(UploadRecord{Device: req["设备"], CurrentTime: req["当前时间"], Phone: req["手机号"], Account: req["账号"], Password: req["密码"]}, s.cfg.Now())
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("save upload: %w", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"消息": "设备 " + req["设备"] + " 已存在相同的账号密码，不会重复保存。",
	})
}

func (s *Server) handleAddEnv(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	record := EnvRecord{
		DeviceCode:       stringFromAny(payload["设备代号"]),
		DeviceID:         stringFromAny(payload["设备ID"]),
		Type:             stringFromAny(payload["类型"]),
		SerialBackupName: stringFromAny(payload["串码备份包名称"]),
		AndroidID:        stringFromAny(payload["安卓ID"]),
		Key:              stringFromAny(payload["密钥"]),
		MaxUsage:         1,
	}
	for field, value := range map[string]string{
		"设备代号":    record.DeviceCode,
		"设备ID":    record.DeviceID,
		"类型":      record.Type,
		"串码备份包名称": record.SerialBackupName,
		"安卓ID":    record.AndroidID,
		"密钥":      record.Key,
	} {
		if value == "" {
			s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "success": false, "msg": field + " is required", "message": field + " is required"})
			return
		}
	}
	id, err := s.cfg.Store.AddEnv(record, s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("add env: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{
		"code":    0,
		"msg":     "添加成功",
		"success": true,
		"message": "添加成功",
		"data": map[string]any{
			"环境id": id,
		},
	})
}

func (s *Server) handleGetEnv(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	record, err := s.cfg.Store.ConsumeEnv(envFilterFromPayload(payload, false), s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get env: %w", err))
		return
	}
	if record == nil {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "success": false, "msg": "暂无可用环境", "message": "暂无可用环境"})
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{
		"code":    0,
		"msg":     "ok",
		"success": true,
		"data":    envData(record),
	})
}

func (s *Server) handleGetEnvEnhanced(w http.ResponseWriter, r *http.Request) {
	s.handleGetEnvWithFilter(w, r, false)
}

func (s *Server) handleGetEnvEnhanced2(w http.ResponseWriter, r *http.Request) {
	s.handleGetEnvWithFilter(w, r, true)
}

func (s *Server) handleGetEnvWithFilter(w http.ResponseWriter, r *http.Request, enhanced2 bool) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	filter := envFilterFromPayload(payload, false)
	if !enhanced2 {
		filter.MinMadeCount = intFromAnyValue(payload["最小制作次数"])
		filter.MaxMadeCount = intFromAnyValue(payload["最大制作次数"])
	}
	record, err := s.cfg.Store.ConsumeEnv(filter, s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get enhanced env: %w", err))
		return
	}
	if record == nil {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "msg": "暂无可用环境", "success": false})
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "msg": "ok", "success": true, "data": envData(record)})
}

func (s *Server) handleGetEnvForMake(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	filter := envFilterFromPayload(payload, false)
	filter.MaxMadeCount = intFromAnyValue(payload["已制作次数"])
	filter.CooldownDays = intFromAnyValue(payload["冷却天数"])
	record, err := s.cfg.Store.ListForMake(filter, s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get env for make: %w", err))
		return
	}
	if record == nil {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "msg": "暂无可制作环境", "success": false})
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "msg": "ok", "success": true, "data": envData(record)})
}

func (s *Server) handleMakeSuccess(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	id, err := int64FromAny(payload["环境id"])
	if err != nil || id <= 0 {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "success": false, "msg": "环境id is required"})
		return
	}
	if err := s.cfg.Store.MarkMakeSuccess(id, s.cfg.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("make success: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "msg": "ok"})
}

func (s *Server) handleIncreaseMakeCount(w http.ResponseWriter, r *http.Request) {
	s.handleAdjustMakeCount(w, r, 1, false)
}
func (s *Server) handleDecreaseMakeCount(w http.ResponseWriter, r *http.Request) {
	s.handleAdjustMakeCount(w, r, -1, false)
}
func (s *Server) handleResetMakeCount(w http.ResponseWriter, r *http.Request) {
	s.handleAdjustMakeCount(w, r, 0, true)
}

func (s *Server) handleAdjustMakeCount(w http.ResponseWriter, r *http.Request, delta int, reset bool) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	id, err := int64FromAny(payload["环境id"])
	if err != nil || id <= 0 {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "success": false, "msg": "环境id is required"})
		return
	}
	if err := s.cfg.Store.AdjustMadeCount(id, delta, reset, s.cfg.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("adjust make count: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "msg": "ok"})
}

func (s *Server) handleGetEnvFixed(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	deviceID := stringFromAny(payload["设备ID"])
	if deviceID == "" {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 1, "success": false, "msg": "设备ID is required"})
		return
	}
	data := map[string]any{
		"设备ID": deviceID, "使用本机设备": true, "环境类型": "QQ888",
		"最大使用次数": 1, "天数限制": 0,
		"最小制作次数": 1, "最大制作次数": 3,
		"最小使用次数": 0, "开始日期": "", "结束日期": "",
		"排序": "创建时间优先",
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": data})
}

func (s *Server) handleStatsByType(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	if _, ok := s.readEncryptedEnvPayload(w, r); !ok {
		return
	}
	stats, err := s.cfg.Store.StatsByType()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": stats})
}

func (s *Server) handleStatsMakeProgress(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	if _, ok := s.readEncryptedEnvPayload(w, r); !ok {
		return
	}
	stats, err := s.cfg.Store.StatsMakeProgress()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": stats})
}

func (s *Server) handleTotal(w http.ResponseWriter, r *http.Request) {
	s.handleScalarStat(w, r, "总数")
}
func (s *Server) handleAvailable(w http.ResponseWriter, r *http.Request) {
	s.handleScalarStat(w, r, "可用")
}
func (s *Server) handleFrozen(w http.ResponseWriter, r *http.Request) {
	s.handleScalarStat(w, r, "冻结")
}
func (s *Server) handleUnused(w http.ResponseWriter, r *http.Request) {
	s.handleScalarStat(w, r, "未使用")
}

func (s *Server) handleScalarStat(w http.ResponseWriter, r *http.Request, name string) {
	if !s.requireEnvStore(w) {
		return
	}
	if _, ok := s.readEncryptedEnvPayload(w, r); !ok {
		return
	}
	stats, err := s.cfg.Store.EnvStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	value := stats.Total
	switch name {
	case "可用":
		value = stats.Available
	case "冻结":
		value = stats.Frozen
	case "未使用":
		value = stats.Unused
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": map[string]any{name: value}})
}

func (s *Server) handleQueryEnvList(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	records, err := s.cfg.Store.ListEnvs(envFilterFromPayload(payload, true))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query env list: %w", err))
		return
	}
	items := make([]map[string]any, 0, len(records))
	for i := range records {
		items = append(items, envData(&records[i]))
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (s *Server) handleQueryEnv(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	id, err := int64FromAny(payload["环境id"])
	if err != nil || id <= 0 {
		records, listErr := s.cfg.Store.ListEnvs(envFilterFromPayload(payload, true))
		if listErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("query env list: %w", listErr))
			return
		}
		items := make([]map[string]any, 0, len(records))
		for i := range records {
			items = append(items, envData(&records[i]))
		}
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": items})
		return
	}
	record, err := s.cfg.Store.GetEnvByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query env: %w", err))
		return
	}
	if record == nil {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": false, "message": "环境不存在"})
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": true, "data": envData(record)})
}

func (s *Server) handleFreezeEnv(w http.ResponseWriter, r *http.Request) {
	s.handleSetEnvFrozen(w, r, true)
}

func (s *Server) handleUnfreezeEnv(w http.ResponseWriter, r *http.Request) {
	s.handleSetEnvFrozen(w, r, false)
}

func (s *Server) handleFreezeByCondition(w http.ResponseWriter, r *http.Request) {
	s.handleSetEnvFrozenByCondition(w, r, true)
}
func (s *Server) handleUnfreezeByCondition(w http.ResponseWriter, r *http.Request) {
	s.handleSetEnvFrozenByCondition(w, r, false)
}

func (s *Server) handleSetEnvFrozenByCondition(w http.ResponseWriter, r *http.Request, frozen bool) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	changed, err := s.cfg.Store.SetEnvFrozenByFilter(envFilterFromPayload(payload, true), frozen, s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("set env frozen by condition: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": map[string]any{"影响数量": changed}})
}

func (s *Server) handleSetEnvFrozen(w http.ResponseWriter, r *http.Request, frozen bool) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	id, err := int64FromAny(payload["环境id"])
	if err != nil || id <= 0 {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": false, "message": "环境id is required"})
		return
	}
	if err := s.cfg.Store.SetEnvFrozen(id, frozen, s.cfg.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("set env frozen: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteEnv(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	id, err := int64FromAny(payload["环境id"])
	if err != nil || id <= 0 {
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": false, "message": "环境id is required"})
		return
	}
	if err := s.cfg.Store.DeleteEnv(id, s.cfg.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete env: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDeleteByCondition(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	changed, err := s.cfg.Store.DeleteEnvsByFilter(envFilterFromPayload(payload, true), s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete env by condition: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"code": 0, "success": true, "data": map[string]any{"删除数量": changed}})
}

func (s *Server) handleCleanEnv(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	if !requirePost(w, r) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	var removed int64
	var err error
	if days, ok := intFromAny(payload["超过天数"]); ok {
		removed, err = s.cfg.Store.CleanEnvOlderThan(*days, s.cfg.Now())
	} else {
		removed, err = s.cfg.Store.CleanEnv()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("clean env: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"清理数量": removed}})
}

func (s *Server) handleQueryByDevice(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	payload, ok := s.readEncryptedEnvPayload(w, r)
	if !ok {
		return
	}
	filter := EnvFilter{DeviceID: stringFromAny(payload["设备ID"])}
	if limit, ok := intFromAny(payload["limit"]); ok {
		filter.Limit = limit
	}
	records, err := s.cfg.Store.ListEnvs(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query by device: %w", err))
		return
	}
	items := make([]map[string]any, 0, len(records))
	for i := range records {
		items = append(items, envData(&records[i]))
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (s *Server) handleEnvStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
		return
	}
	stats, err := s.cfg.Store.EnvStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("env stats: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"总数":  stats.Total,
		"可用":  stats.Available,
		"已消费": stats.Consumed,
		"冻结":  stats.Frozen,
		"已删除": stats.Deleted,
	})
}

func (s *Server) requireEnvStore(w http.ResponseWriter) bool {
	if s.cfg.Store != nil {
		return true
	}
	writeError(w, http.StatusInternalServerError, fmt.Errorf("env store is not configured"))
	return false
}

func (s *Server) readEncryptedEnvPayload(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	if !requirePost(w, r) {
		return nil, false
	}
	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
		return nil, false
	}
	if envelope.Data == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("data is required"))
		return nil, false
	}
	plain, err := decryptDynamicRequestStringAt(envelope.Data, s.cfg.Crypto, s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decrypt env request: %w", err))
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(plain), &payload); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode env payload: %w", err))
		return nil, false
	}
	return payload, true
}

func (s *Server) writeEncryptedEnv(w http.ResponseWriter, status int, body map[string]any) {
	raw, err := json.Marshal(body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("marshal env response: %w", err))
		return
	}
	encrypted, err := encryptResponseStringAt(string(raw), s.cfg.Crypto, s.cfg.Now(), s.cfg.Random)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("encrypt env response: %w", err))
		return
	}
	writeJSON(w, status, map[string]any{"data": encrypted})
}

func envFilterFromPayload(payload map[string]any, includeListFields bool) EnvFilter {
	filter := EnvFilter{
		Type:             stringFromAny(payload["类型"]),
		DeviceCode:       stringFromAny(payload["设备代号"]),
		DeviceID:         stringFromAny(payload["设备ID"]),
		SerialBackupName: stringFromAny(payload["串码备份包名称"]),
		AndroidID:        stringFromAny(payload["安卓ID"]),
		Key:              stringFromAny(payload["密钥"]),
	}
	if maxUsage, ok := intFromAny(payload["最大使用次数"]); ok {
		filter.MaxUsage = maxUsage
	}
	if minUsage, ok := intFromAny(payload["最小使用次数"]); ok {
		filter.MinUsage = minUsage
	}
	if minMade, ok := intFromAny(payload["最小制作次数"]); ok {
		filter.MinMadeCount = minMade
	}
	if maxMade, ok := intFromAny(payload["最大制作次数"]); ok {
		filter.MaxMadeCount = maxMade
	}
	if made, ok := intFromAny(payload["已制作次数"]); ok {
		filter.MinMadeCount = made
	}
	if minDays, ok := intFromAny(payload["最小天数"]); ok {
		filter.MinDays = minDays
	}
	if maxDays, ok := intFromAny(payload["最大天数"]); ok {
		filter.MaxDays = maxDays
	}
	if cooldown, ok := intFromAny(payload["冷却天数"]); ok {
		filter.CooldownDays = cooldown
	}
	filter.Sort = stringFromAny(payload["排序"])
	if olderThanDays, ok := intFromAny(payload["超过天数"]); ok {
		filter.OlderThanDays = olderThanDays
	}
	if includeListFields {
		if frozen, ok := intFromAny(payload["冻结"]); ok {
			filter.Frozen = frozen
		}
		if limit, ok := intFromAny(payload["limit"]); ok {
			filter.Limit = limit
		}
		if offset, ok := intFromAny(payload["offset"]); ok {
			filter.Offset = offset
		}
	}
	return filter
}

func envData(record *EnvRecord) map[string]any {
	data := map[string]any{
		"id":         record.ID,
		"环境id":       record.ID,
		"设备代号":       record.DeviceCode,
		"设备ID":       record.DeviceID,
		"类型":         record.Type,
		"串码备份包名称":    record.SerialBackupName,
		"备份名称":       record.SerialBackupName,
		"安卓ID":       record.AndroidID,
		"密钥":         record.Key,
		"使用次数":       record.UsageCount,
		"最大使用次数":     record.MaxUsage,
		"已制作次数":      record.MadeCount,
		"冻结":         boolInt(record.Frozen),
		"created_at": record.CreatedAt.Format(time.RFC3339),
		"创建时间":       record.CreatedAt.Format(time.RFC3339),
	}
	if record.ConsumedAt != nil {
		data["consumed_at"] = record.ConsumedAt.Format(time.RFC3339)
	}
	if record.DeletedAt != nil {
		data["deleted_at"] = record.DeletedAt.Format(time.RFC3339)
	}
	if record.LastUsedAt != nil {
		data["最后使用时间"] = record.LastUsedAt.Format(time.RFC3339)
	}
	return data
}

func stringFromAny(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intFromAnyValue(value any) *int {
	v, ok := intFromAny(value)
	if !ok {
		return nil
	}
	return v
}
func shanghaiLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return loc
}

func requirePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost {
		return true
	}
	w.Header().Set("Allow", http.MethodPost)
	writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
	return false
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"code":    status,
		"message": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
