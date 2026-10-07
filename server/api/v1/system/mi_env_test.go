package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	serviceSystem "github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMIEnvHandleAcceptsPlainInternalRequest(t *testing.T) {
	useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	payload := map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
		"串码备份包名称": "backup-a", "安卓ID": "android-a", "密钥": "key-a",
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "action", Value: "/add_env"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/internalTool/miEnv/add_env", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	(&MIEnvApi{}).Handle(ctx)

	response := decodePlainMIEnvResponseForTest(t, recorder)
	require.EqualValues(t, 0, response["code"])
	require.Equal(t, "添加成功", response["msg"])
}

func TestMIEnvImportPreservesSourceFieldsAndType(t *testing.T) {
	db := useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	created := int64(1791344655)
	lastUsed := int64(1791344000)
	payload := map[string]any{
		"设备代号": "cepheus", "设备ID": "f54dbf77", "类型": "QQ888",
		"串码备份包名称": "f54dbf77_20260611_014917.dat", "安卓ID": "android-a",
		"密钥": "key-a", "使用次数": 0, "最大使用次数": 1, "已制作次数": 1,
		"冻结": 0, "创建时间": created, "最后使用时间": lastUsed,
	}
	response := dispatchMIEnvForTest(t, "import_env", payload, now)
	require.EqualValues(t, 0, response["code"])
	require.Equal(t, true, response["data"].(map[string]any)["inserted"])
	limit := 10
	items, err := serviceSystem.MIEnvServiceApp.List(serviceSystem.MIEnvFilter{DeviceID: "f54dbf77", Type: "QQ888", Limit: &limit}, now)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "QQ888", items[0].Type)
	require.Equal(t, created, items[0].CreatedAt.Unix())
	require.NotNil(t, items[0].LastUsedAt)
	require.Equal(t, lastUsed, items[0].LastUsedAt.Unix())

	response = dispatchMIEnvForTest(t, "import_env", payload, now.Add(time.Minute))
	require.EqualValues(t, 0, response["code"])
	require.Equal(t, false, response["data"].(map[string]any)["inserted"])
	var count int64
	require.NoError(t, db.Model(&model.SysMIEnvRecord{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestMIEnvBackupProtocolResponses(t *testing.T) {
	useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	added := dispatchMIEnvForTest(t, "add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
		"串码备份包名称": "backup-a", "安卓ID": "android-a", "密钥": "key-a",
	}, now)
	require.EqualValues(t, 0, added["code"])
	addData := added["data"].(map[string]any)
	require.Equal(t, addData["id"], addData["环境id"])
	id := int(addData["id"].(float64))

	reserved := dispatchMIEnvForTest(t, "get_env_for_make", map[string]any{
		"类型": "QQ888", "设备代号": "cepheus", "设备ID": "device-a",
		"已制作次数": 3, "冷却天数": 1,
	}, now)
	require.EqualValues(t, 0, reserved["code"])
	reservedData := reserved["data"].(map[string]any)
	require.EqualValues(t, id, reservedData["id"])
	require.NotEmpty(t, reservedData["制作预约到期时间"])

	duplicate := dispatchMIEnvForTest(t, "get_env_for_make", map[string]any{
		"类型": "QQ888", "设备代号": "cepheus", "设备ID": "device-a",
		"已制作次数": 3, "冷却天数": 1,
	}, now)
	require.EqualValues(t, 1, duplicate["code"])

	made := dispatchMIEnvForTest(t, "make_success", map[string]any{"环境id": id}, now.Add(time.Hour))
	require.EqualValues(t, 0, made["code"])
	require.Equal(t, "制作成功", made["msg"])
	madeData := made["data"].(map[string]any)
	require.EqualValues(t, id, madeData["id"])
	require.EqualValues(t, 2, madeData["已制作次数"])

	queried := dispatchMIEnvForTest(t, "query_env", map[string]any{"环境id": id}, now.Add(time.Hour))
	require.EqualValues(t, 0, queried["code"])
	queriedData := queried["data"].(map[string]any)
	require.EqualValues(t, now.Unix(), queriedData["创建时间"])
	require.EqualValues(t, now.Add(time.Hour).Unix(), queriedData["最后使用时间"])
}

func TestMIEnvMakeSuccessRejectsUnknownOrUnreservedID(t *testing.T) {
	useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	missing := dispatchMIEnvForTest(t, "make_success", map[string]any{"环境id": 999}, now)
	require.EqualValues(t, 1, missing["code"])

	added := dispatchMIEnvForTest(t, "add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
		"串码备份包名称": "backup-a", "安卓ID": "android-a", "密钥": "key-a",
	}, now)
	id := int(added["data"].(map[string]any)["id"].(float64))
	unreserved := dispatchMIEnvForTest(t, "make_success", map[string]any{"环境id": id}, now)
	require.EqualValues(t, 1, unreserved["code"])
}

func TestMIEnvQueryEnvSupportsIDOrFiltersWithoutConsuming(t *testing.T) {
	db := useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	first := dispatchMIEnvForTest(t, "add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ111",
		"串码备份包名称": "backup-a", "安卓ID": "android-a", "密钥": "key-a",
	}, now.Add(-20*24*time.Hour))
	firstID := uint(first["data"].(map[string]any)["id"].(float64))
	second := dispatchMIEnvForTest(t, "add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "device-b", "类型": "QQ111",
		"串码备份包名称": "backup-b", "安卓ID": "android-b", "密钥": "key-b",
	}, now.Add(-20*24*time.Hour))
	secondID := uint(second["data"].(map[string]any)["id"].(float64))
	require.NoError(t, serviceSystem.MIEnvServiceApp.AdjustMadeCount(secondID, 2, false, now.Add(-19*24*time.Hour)))

	filtered := dispatchMIEnvForTest(t, "query_env", map[string]any{
		"类型": "QQ111", "已制作次数": 2, "超过天数": 10,
	}, now)
	require.EqualValues(t, 0, filtered["code"])
	items := filtered["data"].([]any)
	require.Len(t, items, 1)
	require.EqualValues(t, secondID, items[0].(map[string]any)["id"])

	byID := dispatchMIEnvForTest(t, "query_env", map[string]any{"环境id": int(firstID)}, now)
	require.EqualValues(t, 0, byID["code"])
	require.EqualValues(t, firstID, byID["data"].(map[string]any)["id"])

	var records []model.SysMIEnvRecord
	require.NoError(t, db.Order("id").Find(&records).Error)
	require.Len(t, records, 2)
	require.Equal(t, 0, records[0].UsageCount)
	require.Equal(t, 0, records[1].UsageCount)
}

func TestMIEnvQueryEnvSupportsDeviceListLimit(t *testing.T) {
	db := useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	for i := 1; i <= 3; i++ {
		response := dispatchMIEnvForTest(t, "add_env", map[string]any{
			"设备代号": "cepheus", "设备ID": "selected-device", "类型": "QQ111",
			"串码备份包名称": fmt.Sprintf("backup-%d", i),
			"安卓ID":    fmt.Sprintf("android-%d", i),
			"密钥":      fmt.Sprintf("key-%d", i),
		}, now.Add(time.Duration(i)*time.Minute))
		require.EqualValues(t, 0, response["code"])
	}
	dispatchMIEnvForTest(t, "add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "other-device", "类型": "QQ111",
		"串码备份包名称": "backup-other", "安卓ID": "android-other", "密钥": "key-other",
	}, now)

	limited := dispatchMIEnvForTest(t, "query_env", map[string]any{
		"设备ID": "selected-device", "limit": 2,
	}, now)
	require.EqualValues(t, 0, limited["code"])
	require.Equal(t, "ok", limited["msg"])
	require.Len(t, limited["data"].([]any), 2)

	clientRequest := dispatchMIEnvForTest(t, "query_env", map[string]any{
		"设备ID": "selected-device", "limit": 10000,
	}, now)
	require.EqualValues(t, 0, clientRequest["code"])
	require.Equal(t, "ok", clientRequest["msg"])
	clientItems := clientRequest["data"].([]any)
	require.Len(t, clientItems, 3)
	firstClientItem := clientItems[0].(map[string]any)
	require.EqualValues(t, now.Add(time.Minute).Unix(), firstClientItem["创建时间"])
	require.Equal(t, now.Add(time.Minute).Format(time.RFC3339), firstClientItem["created_at"])

	var records []model.SysMIEnvRecord
	require.NoError(t, db.Order("id").Find(&records).Error)
	require.Len(t, records, 4)
	for _, record := range records {
		require.Equal(t, 0, record.UsageCount)
	}
}

func TestMIEnvQueryEnvEncryptedClientContract(t *testing.T) {
	useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Now()

	added := dispatchMIEnvForTest(t, "add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "f54dbf77", "类型": "QQ111",
		"串码备份包名称": "backup-a.dat", "安卓ID": "android-a", "密钥": "key-a",
	}, now)
	require.EqualValues(t, 0, added["code"])

	payload, err := json.Marshal(map[string]any{"设备ID": "f54dbf77", "limit": 10000})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "action", Value: "/query_env"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/internalTool/miEnv/query_env", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	(&MIEnvApi{}).Handle(ctx)

	response := decodePlainMIEnvResponseForTest(t, recorder)
	require.EqualValues(t, 0, response["code"])
	require.Equal(t, true, response["success"])
	require.Equal(t, "ok", response["msg"])

	items, ok := response["data"].([]any)
	require.True(t, ok, "query_env list data must be a JSON array")
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "f54dbf77", item["设备ID"])
	require.Equal(t, "cepheus", item["设备代号"])
	require.Equal(t, "QQ111", item["类型"])
	require.Equal(t, "backup-a.dat", item["串码备份包名称"])
	require.Equal(t, item["串码备份包名称"], item["备份名称"])
	require.Equal(t, "android-a", item["安卓ID"])
	require.Equal(t, "key-a", item["密钥"])
	for _, key := range []string{"id", "环境id", "使用次数", "最大使用次数", "已制作次数", "冻结", "创建时间"} {
		require.IsType(t, float64(0), item[key], "%s must be a JSON number", key)
	}
	require.EqualValues(t, now.Unix(), item["创建时间"])
	require.EqualValues(t, 0, item["最后使用时间"], "never-used records must expose the client sentinel")
	require.IsType(t, "", item["created_at"])

	emptyPayload, err := json.Marshal(map[string]any{"设备ID": "missing-device", "limit": 10000})
	require.NoError(t, err)
	emptyRecorder := httptest.NewRecorder()
	emptyCtx, _ := gin.CreateTestContext(emptyRecorder)
	emptyCtx.Params = gin.Params{{Key: "action", Value: "/query_env"}}
	emptyCtx.Request = httptest.NewRequest(http.MethodPost, "/internalTool/miEnv/query_env", bytes.NewReader(emptyPayload))
	emptyCtx.Request.Header.Set("Content-Type", "application/json")
	(&MIEnvApi{}).Handle(emptyCtx)
	emptyResponse := decodePlainMIEnvResponseForTest(t, emptyRecorder)
	emptyItems, ok := emptyResponse["data"].([]any)
	require.True(t, ok, "empty query_env list data must be [] rather than null or an object")
	require.Empty(t, emptyItems)
}

func TestMIEnvStatsUsesLegacyPlainJSONContract(t *testing.T) {
	useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 10, 7, 4, 30, 0, 0, time.UTC)
	_, err := serviceSystem.MIEnvServiceApp.Add(model.SysMIEnvRecord{
		DeviceCode: "cepheus", DeviceID: "device-a", Type: "QQ111",
		SerialBackupName: "backup-a.dat", AndroidID: "android-a", Key: "key-a",
	}, now)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "action", Value: "/stats"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/internalTool/miEnv/stats", nil)
	(&MIEnvApi{}).Handle(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.EqualValues(t, 1, response["总数"])
	require.EqualValues(t, 1, response["可用"])
	require.EqualValues(t, 0, response["已消费"])
	require.EqualValues(t, 0, response["冻结"])
	require.EqualValues(t, 0, response["已删除"])
	require.NotContains(t, response, "data")
	require.NotContains(t, response, "code")

	postRecorder := httptest.NewRecorder()
	postCtx, _ := gin.CreateTestContext(postRecorder)
	postCtx.Params = gin.Params{{Key: "action", Value: "/stats"}}
	postCtx.Request = httptest.NewRequest(http.MethodPost, "/internalTool/miEnv/stats", bytes.NewBufferString(`{}`))
	(&MIEnvApi{}).Handle(postCtx)
	require.Equal(t, http.StatusMethodNotAllowed, postRecorder.Code)
	require.Equal(t, http.MethodGet, postRecorder.Header().Get("Allow"))
}

func dispatchMIEnvForTest(t *testing.T, action string, payload map[string]any, now time.Time) map[string]any {
	t.Helper()
	cfg := defaultMIEnvCrypto()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	(&MIEnvApi{}).dispatch(ctx, action, payload, cfg, now)
	return decodePlainMIEnvResponseForTest(t, recorder)
}

func decodePlainMIEnvResponseForTest(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	require.Equal(t, 200, recorder.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func useMIEnvAPITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.SysMIEnvRecord{}))
	previous := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previous })
	return db
}
