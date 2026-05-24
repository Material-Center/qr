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
	mux.HandleFunc("/上传", s.handleUpload)
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
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"设备id":    req.DeviceID,
		"开始时间":    now.Format("2006-01-02 15:04"),
		"到期时间":    now.Add(30 * 24 * time.Hour).Format("2006-01-02 15:04:05"),
		"天数":      30,
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
		if _, err := decryptString(value, s.cfg.Crypto); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("decrypt %s: %w", field, err))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"消息": "设备 " + req["设备"] + " 已存在相同的账号密码，不会重复保存。",
	})
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
