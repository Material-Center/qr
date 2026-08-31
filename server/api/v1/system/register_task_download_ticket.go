package system

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

const registerTaskCacheDownloadTicketTTL = 2 * time.Minute

type registerTaskCacheDownloadTicket struct {
	ExporterID uint
	TaskIDs    []uint
	OnlyCache  bool
	ExpiresAt  time.Time
}

var registerTaskCacheDownloadTickets sync.Map

func issueRegisterTaskCacheDownloadTicket(exporterID uint, taskIDs []uint, onlyCache bool) (string, time.Time, error) {
	if exporterID == 0 || len(taskIDs) == 0 {
		return "", time.Time{}, errors.New("invalid download ticket payload")
	}
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", time.Time{}, err
	}
	ticket := base64.RawURLEncoding.EncodeToString(randomBytes)
	expiresAt := time.Now().Add(registerTaskCacheDownloadTicketTTL)
	registerTaskCacheDownloadTickets.Range(func(key, value any) bool {
		if payload, ok := value.(registerTaskCacheDownloadTicket); ok && time.Now().After(payload.ExpiresAt) {
			registerTaskCacheDownloadTickets.Delete(key)
		}
		return true
	})
	registerTaskCacheDownloadTickets.Store(ticket, registerTaskCacheDownloadTicket{
		ExporterID: exporterID,
		TaskIDs:    append([]uint(nil), taskIDs...),
		OnlyCache:  onlyCache,
		ExpiresAt:  expiresAt,
	})
	return ticket, expiresAt, nil
}

func consumeRegisterTaskCacheDownloadTicket(raw string) (registerTaskCacheDownloadTicket, error) {
	ticket := strings.TrimSpace(raw)
	if ticket == "" {
		return registerTaskCacheDownloadTicket{}, errors.New("下载凭证不能为空")
	}
	value, ok := registerTaskCacheDownloadTickets.LoadAndDelete(ticket)
	if !ok {
		return registerTaskCacheDownloadTicket{}, errors.New("下载凭证无效或已使用")
	}
	payload, ok := value.(registerTaskCacheDownloadTicket)
	if !ok || time.Now().After(payload.ExpiresAt) {
		return registerTaskCacheDownloadTicket{}, errors.New("下载凭证已过期")
	}
	return payload, nil
}

// DownloadRegisterTaskCacheByTicket lets the browser download a real HTTP
// attachment. This avoids blob URL downloads, which are unreliable in mobile
// Safari and embedded Android WebViews.
func (a *RegisterTaskApi) DownloadRegisterTaskCacheByTicket(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	payload, err := consumeRegisterTaskCacheDownloadTicket(c.Query("ticket"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	writeRegisterTaskCache(c, payload.ExporterID, payload.TaskIDs, payload.OnlyCache)
}
