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

func TestRegisterTaskSummaryFallsBackToPromoterLeader(t *testing.T) {
	setupRegisterTaskSummaryTestDB(t)

	now := time.Now()
	leaderID := uint(2)
	promoterID := uint(3)
	successCode := 0
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: leaderID},
		Username:    "leader",
		NickName:    "团长",
		AuthorityId: 200,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: promoterID},
		Username:    "promoter",
		NickName:    "地推",
		AuthorityId: 300,
		LeaderID:    &leaderID,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterTask{
		Phone:        "18800000001",
		PromoterID:   promoterID,
		StatusCode:   &successCode,
		QQLoggedList: `["10001"]`,
		FinishedAt:   &now,
		ExpiresAt:    now.Add(time.Hour),
	}).Error)

	adminGot, err := (&RegisterTaskService{}).GetSummary(roleAdmin, 100, modelSystemReq.RegisterTaskSummaryFilter{})
	require.NoError(t, err)
	require.Len(t, adminGot.Leaders, 1)
	require.Equal(t, leaderID, adminGot.Leaders[0].LeaderID)
	require.Equal(t, "团长", adminGot.Leaders[0].LeaderName)
	require.EqualValues(t, 1, adminGot.Leaders[0].SuccessCount)
	require.Len(t, adminGot.Promoters, 1)
	require.Equal(t, leaderID, adminGot.Promoters[0].LeaderID)
	require.Equal(t, promoterID, adminGot.Promoters[0].PromoterID)

	leaderGot, err := (&RegisterTaskService{}).GetSummary(roleLeader, leaderID, modelSystemReq.RegisterTaskSummaryFilter{})
	require.NoError(t, err)
	require.Len(t, leaderGot.Leaders, 1)
	require.Len(t, leaderGot.Promoters, 1)
}

func TestRegisterTaskListFallsBackToPromoterLeader(t *testing.T) {
	setupRegisterTaskSummaryTestDB(t)

	now := time.Now()
	leaderID := uint(2)
	promoterID := uint(3)
	successCode := 0
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: leaderID},
		Username:    "leader",
		NickName:    "团长",
		AuthorityId: 200,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysUser{
		GVA_MODEL:   global.GVA_MODEL{ID: promoterID},
		Username:    "promoter",
		NickName:    "地推",
		AuthorityId: 300,
		LeaderID:    &leaderID,
		Enable:      1,
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&modelSystem.SysRegisterTask{
		Phone:        "18800000001",
		PromoterID:   promoterID,
		StatusCode:   &successCode,
		QQLoggedList: `["10001"]`,
		FinishedAt:   &now,
		ExpiresAt:    now.Add(time.Hour),
	}).Error)

	adminGot, err := (&RegisterTaskService{}).GetTaskList(roleAdmin, 100, modelSystemReq.RegisterTaskList{})
	require.NoError(t, err)
	require.Len(t, adminGot.List, 1)
	require.Equal(t, leaderID, adminGot.List[0].Leader.ID)
	require.Equal(t, "团长", adminGot.List[0].Leader.NickName)

	leaderGot, err := (&RegisterTaskService{}).GetTaskList(roleLeader, leaderID, modelSystemReq.RegisterTaskList{})
	require.NoError(t, err)
	require.Len(t, leaderGot.List, 1)
	require.Equal(t, leaderID, leaderGot.List[0].Leader.ID)
}

func TestRegisterTaskDeputySummaryOnlyIncludesCreatedPromoters(t *testing.T) {
	setupRegisterTaskSummaryTestDB(t)

	now := time.Now()
	leaderID := uint(2)
	deputyID := uint(20)
	successCode := 0
	require.NoError(t, global.GVA_DB.Create(&[]modelSystem.SysUser{
		{GVA_MODEL: global.GVA_MODEL{ID: leaderID}, Username: "leader", NickName: "团长", AuthorityId: 200, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: deputyID}, Username: "deputy", NickName: "副团长", AuthorityId: 210, LeaderID: &leaderID, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 31}, Username: "own-promoter", NickName: "直属地推", AuthorityId: 300, LeaderID: &leaderID, CreatedBy: deputyID, Enable: 1},
		{GVA_MODEL: global.GVA_MODEL{ID: 32}, Username: "other-promoter", NickName: "其他地推", AuthorityId: 300, LeaderID: &leaderID, CreatedBy: 99, Enable: 1},
	}).Error)
	require.NoError(t, global.GVA_DB.Create(&[]modelSystem.SysRegisterTask{
		{Phone: "18800000001", PromoterID: 31, LeaderID: &leaderID, StatusCode: &successCode, QQLoggedList: `["10001"]`, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
		{Phone: "18800000002", PromoterID: 32, LeaderID: &leaderID, StatusCode: &successCode, QQLoggedList: `["10002"]`, FinishedAt: &now, ExpiresAt: now.Add(time.Hour)},
	}).Error)

	summary, err := (&RegisterTaskService{}).GetSummary(roleDeputyLeader, deputyID, modelSystemReq.RegisterTaskSummaryFilter{})
	require.NoError(t, err)
	require.Len(t, summary.Leaders, 1)
	require.Len(t, summary.Promoters, 1)
	require.Equal(t, uint(31), summary.Promoters[0].PromoterID)

	list, err := (&RegisterTaskService{}).GetTaskList(roleDeputyLeader, deputyID, modelSystemReq.RegisterTaskList{})
	require.NoError(t, err)
	require.Len(t, list.List, 1)
	require.Equal(t, uint(31), list.List[0].PromoterID)
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
