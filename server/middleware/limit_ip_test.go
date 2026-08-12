package middleware

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLimitWithTimeSkipsWhenLimitDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark: func(key string, expire int, limit int) error {
			called = true
			return errors.New("should not be called")
		},
		Expire: 60,
		Limit:  0,
	}.LimitWithTime())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
	require.False(t, called)
}

func TestLimitWithTimeLogsClientIPWhenLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, logs := observer.New(zap.ErrorLevel)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.New(core)
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark:   func(string, int, int) error { return errors.New("limited") },
		Expire:        60,
		Limit:         1,
	}.LimitWithTime())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "203.0.113.8:12345"
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, logs.All(), 1)
	entry := logs.All()[0]
	require.Equal(t, "limit", entry.Message)
	require.Equal(t, "203.0.113.8", entry.ContextMap()["ip"])
}

func TestLimitWithTimeSkipsDeviceTaskOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	called := false
	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark: func(string, int, int) error {
			called = true
			return errors.New("limited")
		},
		Expire: 60,
		Limit:  1,
	}.LimitWithTime())
	router.POST("/phoneRegisterTask/open-api/task", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/task", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
	require.False(t, called)
}

func TestLimitWithTimeStillLimitsPromoterOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark:   func(string, int, int) error { return errors.New("limited") },
		Expire:        60,
		Limit:         1,
	}.LimitWithTime())
	router.POST("/phoneRegisterTask/open-api/promoter/task", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/promoter/task", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "limited", body["msg"])
}

func TestLimitWithTimeStillLimitsOtherOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLogger := global.GVA_LOG
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_LOG = previousLogger
	})

	router := gin.New()
	router.Use(LimitConfig{
		GenerationKey: func(c *gin.Context) string { return c.ClientIP() },
		CheckOrMark:   func(string, int, int) error { return errors.New("limited") },
		Expire:        60,
		Limit:         1,
	}.LimitWithTime())
	router.POST("/phoneRegisterTask/open-api/other", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/phoneRegisterTask/open-api/other", nil)
	router.ServeHTTP(rec, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "limited", body["msg"])
}

func TestDefaultLoginLimitUsesDedicatedConfigAndKey(t *testing.T) {
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.System.LoginLimitCountIP = 8
	global.GVA_CONFIG.System.LoginLimitTimeIP = 300

	limit := DefaultLoginLimitConfig()

	require.Equal(t, 8, limit.Limit)
	require.Equal(t, 300, limit.Expire)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/base/login", nil)
	c.Request.RemoteAddr = "192.0.2.10:12345"
	require.Equal(t, "GVA_Login_Limit192.0.2.10", limit.GenerationKey(c))
}

func TestSetLimitWithTimeRepairsExistingKeyWithoutTTL(t *testing.T) {
	server := newLimitFakeRedisServer(t, map[string]limitFakeRedisValue{
		"GVA_Limit203.0.113.9": {value: "3", ttl: -1},
	})
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: server.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		server.close()
	})

	require.NoError(t, SetLimitWithTime("GVA_Limit203.0.113.9", 10, time.Minute))

	ttl, err := global.GVA_REDIS.TTL(context.Background(), "GVA_Limit203.0.113.9").Result()
	require.NoError(t, err)
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, time.Minute)
}

func TestLoginUserAgentGuardRejectsScriptUAWithGenericFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	router := gin.New()
	router.Use(LoginUserAgentGuard())
	router.POST("/base/login", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	req.Header.Set("User-Agent", "curl/8.0.1")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, called)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "用户名不存在或者密码错误", body["msg"])
}

func TestLoginUserAgentGuardAllowsBrowserUA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(LoginUserAgentGuard())
	router.POST("/base/login", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}

type limitFakeRedisValue struct {
	value string
	ttl   time.Duration
}

type limitFakeRedisServer struct {
	addr   string
	ln     net.Listener
	mu     sync.Mutex
	values map[string]limitFakeRedisValue
}

func newLimitFakeRedisServer(t *testing.T, values map[string]limitFakeRedisValue) *limitFakeRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &limitFakeRedisServer{
		addr:   ln.Addr().String(),
		ln:     ln,
		values: values,
	}
	go s.serve()
	return s
}

func (s *limitFakeRedisServer) close() {
	_ = s.ln.Close()
}

func (s *limitFakeRedisServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *limitFakeRedisServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	queued := false
	for {
		args, err := readLimitRESPArray(reader)
		if err != nil {
			return
		}
		if len(args) == 0 {
			_, _ = conn.Write([]byte("-ERR empty command\r\n"))
			continue
		}
		cmd := strings.ToLower(args[0])
		switch cmd {
		case "hello":
			_, _ = conn.Write([]byte("%7\r\n+server\r\n+redis\r\n+version\r\n+7.0.0\r\n+proto\r\n:3\r\n+id\r\n:1\r\n+mode\r\n+standalone\r\n+role\r\n+master\r\n+modules\r\n*0\r\n"))
		case "exists":
			s.mu.Lock()
			_, ok := s.values[args[1]]
			s.mu.Unlock()
			if ok {
				_, _ = conn.Write([]byte(":1\r\n"))
			} else {
				_, _ = conn.Write([]byte(":0\r\n"))
			}
		case "get":
			s.mu.Lock()
			v, ok := s.values[args[1]]
			s.mu.Unlock()
			if !ok {
				_, _ = conn.Write([]byte("$-1\r\n"))
				continue
			}
			_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v.value), v.value)
		case "incr":
			if queued {
				_, _ = conn.Write([]byte("+QUEUED\r\n"))
				continue
			}
			s.mu.Lock()
			v := s.values[args[1]]
			v.value = "4"
			s.values[args[1]] = v
			s.mu.Unlock()
			_, _ = conn.Write([]byte(":4\r\n"))
		case "expire":
			if queued {
				_, _ = conn.Write([]byte("+QUEUED\r\n"))
				continue
			}
			s.mu.Lock()
			v := s.values[args[1]]
			v.ttl = time.Minute
			s.values[args[1]] = v
			s.mu.Unlock()
			_, _ = conn.Write([]byte(":1\r\n"))
		case "pttl":
			s.mu.Lock()
			v, ok := s.values[args[1]]
			s.mu.Unlock()
			if !ok {
				_, _ = conn.Write([]byte(":-2\r\n"))
				continue
			}
			if v.ttl < 0 {
				_, _ = conn.Write([]byte(":-1\r\n"))
				continue
			}
			_, _ = fmt.Fprintf(conn, ":%d\r\n", v.ttl.Milliseconds())
		case "ttl":
			s.mu.Lock()
			v, ok := s.values[args[1]]
			s.mu.Unlock()
			if !ok {
				_, _ = conn.Write([]byte(":-2\r\n"))
				continue
			}
			if v.ttl < 0 {
				_, _ = conn.Write([]byte(":-1\r\n"))
				continue
			}
			_, _ = fmt.Fprintf(conn, ":%d\r\n", int(v.ttl.Seconds()))
		case "multi":
			queued = true
			_, _ = conn.Write([]byte("+OK\r\n"))
		case "exec":
			queued = false
			_, _ = conn.Write([]byte("*2\r\n:4\r\n:1\r\n"))
		case "client":
			_, _ = conn.Write([]byte("+OK\r\n"))
		default:
			_, _ = conn.Write([]byte("+OK\r\n"))
		}
	}
}

func readLimitRESPArray(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected RESP line %q", line)
	}
	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(line), "*%d", &count); err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		bulkHeader, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		var length int
		if _, err := fmt.Sscanf(strings.TrimSpace(bulkHeader), "$%d", &length); err != nil {
			return nil, err
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:length]))
	}
	return args, nil
}
