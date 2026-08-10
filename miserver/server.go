package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ServerConfig struct {
	Crypto CryptoConfig
	Now    func() time.Time
	Random io.Reader
	Store  *Store

	LogOutput io.Writer
}

type Server struct {
	cfg ServerConfig
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
	return s.accessLog(mux)
}

func (s *Server) AuthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/shanghaitime", s.handleShanghaiTime)
	mux.HandleFunc("/get_device", s.handleGetDevice)
	mux.HandleFunc("/use_code", s.handleUseCode)
	mux.HandleFunc("/stoptime", s.handleStopTime)
	return s.accessLog(mux)
}

func (s *Server) UploadHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/上传", s.handleUpload)
	return s.accessLog(mux)
}

func (s *Server) EnvHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/add_env", s.handleAddEnv)
	mux.HandleFunc("/get_env", s.handleGetEnv)
	mux.HandleFunc("/query_env_list", s.handleQueryEnvList)
	mux.HandleFunc("/query_env", s.handleQueryEnv)
	mux.HandleFunc("/freeze_env", s.handleFreezeEnv)
	mux.HandleFunc("/unfreeze_env", s.handleUnfreezeEnv)
	mux.HandleFunc("/delete_env", s.handleDeleteEnv)
	mux.HandleFunc("/clean_env", s.handleCleanEnv)
	mux.HandleFunc("/query_by_device", s.handleQueryByDevice)
	mux.HandleFunc("/stats", s.handleEnvStats)
	return s.accessLog(mux)
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(lw, r)

		fmt.Fprintf(
			s.cfg.LogOutput,
			"%s %s %d %dB %s\n",
			r.Method,
			r.URL.RequestURI(),
			lw.status,
			lw.bytes,
			time.Since(start).Round(time.Microsecond),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (s *Server) handleShanghaiTime(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r) {
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
	if !requirePost(w, r) {
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode json: %w", err))
		return
	}
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("device_id is required"))
		return
	}

	now := s.cfg.Now().In(shanghaiLocation())
	authorizedUntil := now.Add(30 * 24 * time.Hour)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"设备id":    req.DeviceID,
		"开始时间":    now.Format("2006-01-02 15:04"),
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
			s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": false, "message": field + " is required"})
			return
		}
	}
	id, err := s.cfg.Store.AddEnv(record, s.cfg.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("add env: %w", err))
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{
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
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": false, "message": "暂无可用环境"})
		return
	}
	s.writeEncryptedEnv(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    envData(record),
	})
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
		s.writeEncryptedEnv(w, http.StatusOK, map[string]any{"success": false, "message": "环境id is required"})
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

func (s *Server) handleCleanEnv(w http.ResponseWriter, r *http.Request) {
	if !s.requireEnvStore(w) {
		return
	}
	if !requirePost(w, r) {
		return
	}
	if _, ok := s.readEncryptedEnvPayload(w, r); !ok {
		return
	}
	removed, err := s.cfg.Store.CleanEnv()
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
		"冻结":         boolInt(record.Frozen),
		"created_at": record.CreatedAt.Format(time.RFC3339),
	}
	if record.ConsumedAt != nil {
		data["consumed_at"] = record.ConsumedAt.Format(time.RFC3339)
	}
	if record.DeletedAt != nil {
		data["deleted_at"] = record.DeletedAt.Format(time.RFC3339)
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
