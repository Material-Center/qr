package middleware

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

const (
	requestSignatureTimestampHeader = "X-Req-Timestamp"
	requestSignatureNonceHeader     = "X-Req-Nonce"
	requestSignatureHeader          = "X-Req-Signature"
	requestSignatureMaxSkew         = 300
	requestSignatureNoncePrefix     = "GVA_Req_Sign_Nonce:"
	requestSignatureSalt            = "qr-web-request-signature-v1"
)

var requestSignatureNonceMarker = markRequestSignatureNonce

func RequestSignatureGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			failRequestSignature(c)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		timestamp := c.GetHeader(requestSignatureTimestampHeader)
		nonce := c.GetHeader(requestSignatureNonceHeader)
		signature := c.GetHeader(requestSignatureHeader)
		if !validRequestSignatureHeaders(timestamp, nonce, signature) {
			failRequestSignature(c)
			return
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil || !requestSignatureTimestampInWindow(ts) {
			failRequestSignature(c)
			return
		}

		expected := buildRequestSignature(c.Request.Method, c.Request.URL.Path, timestamp, nonce, body)
		actual, err := hex.DecodeString(signature)
		if err != nil || !bytes.Equal(expected, actual) {
			failRequestSignature(c)
			return
		}

		if err := requestSignatureNonceMarker(nonce, requestSignatureNonceTTL()); err != nil {
			failRequestSignature(c)
			return
		}

		c.Next()
	}
}

func failRequestSignature(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": response.ERROR, "msg": "用户名不存在或者密码错误"})
	c.Abort()
}

func validRequestSignatureHeaders(timestamp, nonce, signature string) bool {
	return timestamp != "" && len(nonce) >= 8 && len(nonce) <= 128 && len(signature) == md5.Size*2
}

func requestSignatureTimestampInWindow(timestamp int64) bool {
	now := time.Now().UnixMilli()
	skew := int64(requestSignatureMaxSkew) * int64(time.Second/time.Millisecond)
	return timestamp >= now-skew && timestamp <= now+skew
}

func requestSignatureNonceTTL() time.Duration {
	return time.Duration(requestSignatureMaxSkew*2) * time.Second
}

func buildRequestSignature(method, path, timestamp, nonce string, body []byte) []byte {
	ts, _ := strconv.ParseInt(timestamp, 10, 64)
	sum := md5.Sum([]byte(deriveRequestSignatureKey(time.UnixMilli(ts)) + "\n" +
		strings.ToUpper(method) + "\n" +
		path + "\n" +
		timestamp + "\n" +
		nonce + "\n" +
		string(body)))
	return sum[:]
}

func deriveRequestSignatureKey(t time.Time) string {
	sum := md5.Sum([]byte(t.UTC().Format("20060102") + requestSignatureSalt))
	return hex.EncodeToString(sum[:])
}

func markRequestSignatureNonce(nonce string, ttl time.Duration) error {
	if global.GVA_REDIS == nil {
		return nil
	}
	ok, err := global.GVA_REDIS.SetNX(context.Background(), requestSignatureNoncePrefix+nonce, "1", ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("request signature nonce replayed")
	}
	return nil
}
