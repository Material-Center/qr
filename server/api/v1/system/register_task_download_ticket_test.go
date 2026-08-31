package system

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRegisterTaskCacheDownloadTicketIsSingleUse(t *testing.T) {
	ticket, expiresAt, err := issueRegisterTaskCacheDownloadTicket(7, []uint{3, 5}, true)
	require.NoError(t, err)
	require.NotEmpty(t, ticket)
	require.WithinDuration(t, time.Now().Add(registerTaskCacheDownloadTicketTTL), expiresAt, time.Second)

	payload, err := consumeRegisterTaskCacheDownloadTicket(ticket)
	require.NoError(t, err)
	require.Equal(t, uint(7), payload.ExporterID)
	require.Equal(t, []uint{3, 5}, payload.TaskIDs)
	require.True(t, payload.OnlyCache)

	_, err = consumeRegisterTaskCacheDownloadTicket(ticket)
	require.EqualError(t, err, "下载凭证无效或已使用")
}

func TestRegisterTaskCacheDownloadTicketRejectsExpiredTicket(t *testing.T) {
	const ticket = "expired-ticket"
	registerTaskCacheDownloadTickets.Store(ticket, registerTaskCacheDownloadTicket{
		ExporterID: 7,
		TaskIDs:    []uint{3},
		ExpiresAt:  time.Now().Add(-time.Second),
	})

	_, err := consumeRegisterTaskCacheDownloadTicket(ticket)
	require.EqualError(t, err, "下载凭证已过期")
	_, exists := registerTaskCacheDownloadTickets.Load(ticket)
	require.False(t, exists)
}
