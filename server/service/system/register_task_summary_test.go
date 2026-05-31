package system

import (
	"fmt"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	modelSystem "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	modelSystemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	modelSystemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRegisterTaskSummaryTestDB(t *testing.T) {
	t.Helper()
	global.GVA_REDIS = nil
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&modelSystem.SysUser{},
		&modelSystem.SysRegisterTask{},
	))
	global.GVA_DB = db
}

func TestRegisterTaskSummaryOrderIsStable(t *testing.T) {
	setupRegisterTaskSummaryTestDB(t)

	now := time.Now()
	successCode := 0
	leaderIDs := []uint{30, 10, 20}
	for _, leaderID := range leaderIDs {
		require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
			GVA_MODEL:   global.GVA_MODEL{ID: leaderID},
			Username:    "leader",
			NickName:    "团长",
			AuthorityId: 200,
			Enable:      1,
		}).Error)
		for _, promoterID := range []uint{leaderID + 2, leaderID + 1} {
			require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterTask{
				Phone:        fmt.Sprintf("188%08d", promoterID),
				PromoterID:   promoterID,
				LeaderID:     &leaderID,
				StatusCode:   &successCode,
				QQLoggedList: `["10001"]`,
				FinishedAt:   &now,
				ExpiresAt:    now.Add(time.Hour),
			}).Error)
		}
	}

	for i := 0; i < 20; i++ {
		got, err := (&RegisterTaskService{}).GetSummary(roleAdmin, 100, modelSystemReq.RegisterTaskSummaryFilter{})
		require.NoError(t, err)
		require.Equal(t, []uint{10, 20, 30}, registerSummaryLeaderIDs(got.Leaders))
		require.Equal(t, []uint{11, 12, 21, 22, 31, 32}, registerSummaryPromoterIDs(got.Promoters))
	}
}

func registerSummaryLeaderIDs(items []modelSystemRes.RegisterTaskSummaryItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.LeaderID)
	}
	return ids
}

func registerSummaryPromoterIDs(items []modelSystemRes.RegisterTaskSummaryItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.PromoterID)
	}
	return ids
}
