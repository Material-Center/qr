package middleware

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestSignatureGuardAllowsValidSignatureAndRestoresBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldMarker := requestSignatureNonceMarker
	requestSignatureNonceMarker = func(nonce string, ttl time.Duration) error { return nil }
	defer func() { requestSignatureNonceMarker = oldMarker }()

	payload := `{"username":"admin","password":"123456"}`
	timestamp := time.Now().UnixMilli()
	nonce := "nonce-123456"
	signature := testRequestSignature(http.MethodPost, "/base/login", timestamp, nonce, payload)

	router := gin.New()
	router.Use(RequestSignatureGuard())
	router.POST("/base/login", func(c *gin.Context) {
		body, err := c.GetRawData()
		require.NoError(t, err)
		require.Equal(t, payload, string(body))
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", strings.NewReader(payload))
	req.Header.Set("X-Req-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Req-Nonce", nonce)
	req.Header.Set("X-Req-Signature", signature)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Body.String())
}

func TestRequestSignatureGuardRejectsMissingSignatureWithGenericFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	router := gin.New()
	router.Use(RequestSignatureGuard())
	router.POST("/base/login", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, called)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, float64(7), body["code"])
	require.Equal(t, "用户名不存在或者密码错误", body["msg"])
}

func TestRequestSignatureGuardRejectsExpiredTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	payload := `{"username":"admin","password":"123456"}`
	timestamp := time.Now().Add(-10 * time.Minute).UnixMilli()
	nonce := "nonce-123456"
	signature := testRequestSignature(http.MethodPost, "/base/login", timestamp, nonce, payload)

	called := false
	router := gin.New()
	router.Use(RequestSignatureGuard())
	router.POST("/base/login", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", strings.NewReader(payload))
	req.Header.Set("X-Req-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Req-Nonce", nonce)
	req.Header.Set("X-Req-Signature", signature)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, called)
}

func TestRequestSignatureGuardRejectsReplayedNonce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldMarker := requestSignatureNonceMarker
	requestSignatureNonceMarker = func(nonce string, ttl time.Duration) error { return errors.New("replay") }
	defer func() { requestSignatureNonceMarker = oldMarker }()

	payload := `{"username":"admin","password":"123456"}`
	timestamp := time.Now().UnixMilli()
	nonce := "nonce-123456"
	signature := testRequestSignature(http.MethodPost, "/base/login", timestamp, nonce, payload)

	called := false
	router := gin.New()
	router.Use(RequestSignatureGuard())
	router.POST("/base/login", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/base/login", strings.NewReader(payload))
	req.Header.Set("X-Req-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Req-Nonce", nonce)
	req.Header.Set("X-Req-Signature", signature)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, called)
}

func testRequestSignature(method, path string, timestamp int64, nonce string, payload string) string {
	sum := md5.Sum([]byte(deriveRequestSignatureKey(time.UnixMilli(timestamp)) + "\n" +
		strings.ToUpper(method) + "\n" +
		path + "\n" +
		strconv.FormatInt(timestamp, 10) + "\n" +
		nonce + "\n" +
		payload))
	return hex.EncodeToString(sum[:])
}
