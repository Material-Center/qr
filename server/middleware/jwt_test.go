package middleware

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	serviceSystem "github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupJWTAuthTokenPurposeTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	global.GVA_CONFIG = config.Server{}
	global.GVA_CONFIG.JWT.SigningKey = "jwt-auth-token-purpose-test-key"
	global.GVA_CONFIG.JWT.BufferTime = "1d"
	global.GVA_CONFIG.JWT.ExpiresTime = "7d"
	global.GVA_CONFIG.JWT.Issuer = "test"
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(7 * 24 * time.Hour))
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysAuthority{},
		&modelSystem.SysApiToken{},
		&modelSystem.JwtBlacklist{},
	))
	global.GVA_DB = db
}

func newJWTAuthTestRouter() *gin.Engine {
	router := gin.New()
	router.GET("/private", JWTAuth(), func(c *gin.Context) {
		response.OkWithMessage("ok", c)
	})
	return router
}

func TestJWTAuthRejectsOpenAPIToken(t *testing.T) {
	setupJWTAuthTokenPurposeTest(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "promoter",
		NickName:    "promoter",
		AuthorityId: 300,
		Enable:      1,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)
	token, err := (&serviceSystem.ApiTokenService{}).CreateApiTokenForOperator(100, modelSystem.SysApiToken{
		UserID:      user.ID,
		AuthorityID: user.AuthorityId,
	}, 30)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("x-token", token)
	rec := httptest.NewRecorder()
	newJWTAuthTestRouter().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Contains(t, got.Msg, "OpenAPI")
}

func TestJWTAuthAcceptsLoginToken(t *testing.T) {
	setupJWTAuthTokenPurposeTest(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "promoter",
		NickName:    "promoter",
		AuthorityId: 300,
		Enable:      1,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)
	token, _, err := utils.LoginToken(&user)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("x-token", token)
	rec := httptest.NewRecorder()
	newJWTAuthTestRouter().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuthRejectsDisabledUserWithExistingLoginToken(t *testing.T) {
	setupJWTAuthTokenPurposeTest(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "disabled-promoter",
		NickName:    "disabled promoter",
		AuthorityId: 300,
		Enable:      2,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)
	token, _, err := utils.LoginToken(&user)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("x-token", token)
	rec := httptest.NewRecorder()
	newJWTAuthTestRouter().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	var got response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Contains(t, got.Msg, "用户被禁用")
}

func TestJWTAuthRejectsUserDisabledInRedisCache(t *testing.T) {
	setupJWTAuthTokenPurposeTest(t)
	user := modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: 1},
		UUID:        uuid.New(),
		Username:    "cached-disabled-promoter",
		NickName:    "cached disabled promoter",
		AuthorityId: 300,
		Enable:      1,
	}
	require.NoError(t, global.GVA_DB.Create(&user).Error)
	redisServer := newJWTAuthFakeRedisServer(t, map[string]string{
		fmt.Sprintf("login:user:enable:%s", user.UUID.String()): "2",
	})
	originalRedis := global.GVA_REDIS
	global.GVA_REDIS = redis.NewClient(&redis.Options{Addr: redisServer.addr, Protocol: 2})
	t.Cleanup(func() {
		_ = global.GVA_REDIS.Close()
		global.GVA_REDIS = originalRedis
		redisServer.close()
	})
	token, _, err := utils.LoginToken(&user)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("x-token", token)
	rec := httptest.NewRecorder()
	newJWTAuthTestRouter().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, 0, redisServer.count("set"))
}

type jwtAuthFakeRedisServer struct {
	addr   string
	ln     net.Listener
	values map[string]string
	counts chan string
}

func newJWTAuthFakeRedisServer(t *testing.T, values map[string]string) *jwtAuthFakeRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &jwtAuthFakeRedisServer{
		addr:   ln.Addr().String(),
		ln:     ln,
		values: values,
		counts: make(chan string, 16),
	}
	go s.serve()
	return s
}

func (s *jwtAuthFakeRedisServer) close() {
	_ = s.ln.Close()
}

func (s *jwtAuthFakeRedisServer) count(command string) int {
	total := 0
	for {
		select {
		case got := <-s.counts:
			if got == command {
				total++
			}
		default:
			return total
		}
	}
}

func (s *jwtAuthFakeRedisServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *jwtAuthFakeRedisServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		args, err := readJWTAuthRESPArray(reader)
		if err != nil {
			return
		}
		if len(args) == 0 {
			_, _ = conn.Write([]byte("-ERR empty command\r\n"))
			continue
		}
		cmd := strings.ToLower(args[0])
		select {
		case s.counts <- cmd:
		default:
		}
		switch cmd {
		case "hello":
			_, _ = conn.Write([]byte("%7\r\n+server\r\n+redis\r\n+version\r\n+7.0.0\r\n+proto\r\n:3\r\n+id\r\n:1\r\n+mode\r\n+standalone\r\n+role\r\n+master\r\n+modules\r\n*0\r\n"))
		case "get":
			value, ok := s.values[args[1]]
			if !ok {
				_, _ = conn.Write([]byte("$-1\r\n"))
				continue
			}
			_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(value), value)
		case "set", "client":
			_, _ = conn.Write([]byte("+OK\r\n"))
		default:
			_, _ = conn.Write([]byte("+OK\r\n"))
		}
	}
}

func readJWTAuthRESPArray(reader *bufio.Reader) ([]string, error) {
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
