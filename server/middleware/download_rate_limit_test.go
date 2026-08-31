package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDownloadRateLimitRejectsBurstOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	downloadRateLimiters = sync.Map{}
	downloadSlots = make(chan struct{}, downloadMaxConcurrency)

	router := gin.New()
	router.GET("/download", DownloadRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/download", nil)
		request.RemoteAddr = "192.0.2.1:1234"
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusNoContent, recorder.Code)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/download", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "2", recorder.Header().Get("Retry-After"))
}
