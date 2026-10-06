package system

import (
	"bytes"
	"encoding/base64"
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

func TestMIEnvHandleAcceptsCompiledClientEncryption(t *testing.T) {
	useMIEnvAPITestDB(t)
	gin.SetMode(gin.TestMode)
	now := time.Now()
	payload := map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
		"串码备份包名称": "backup-a", "安卓ID": "android-a", "密钥": "key-a",
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	envelopeRaw, err := json.Marshal(map[string]string{"data": encryptMIEnvRequestForTest(t, raw, defaultMIEnvCrypto(), now)})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "action", Value: "/add_env"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/internalTool/miEnv/add_env", bytes.NewReader(envelopeRaw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	(&MIEnvApi{}).Handle(ctx)

	response := decodeMIEnvResponseForTest(t, recorder, defaultMIEnvCrypto(), now)
	require.EqualValues(t, 0, response["code"])
	require.Equal(t, "添加成功", response["msg"])
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
	require.Len(t, limited["data"].([]any), 2)

	clientRequest := dispatchMIEnvForTest(t, "query_env", map[string]any{
		"设备ID": "selected-device", "limit": 10000,
	}, now)
	require.EqualValues(t, 0, clientRequest["code"])
	require.Len(t, clientRequest["data"].([]any), 3)

	var records []model.SysMIEnvRecord
	require.NoError(t, db.Order("id").Find(&records).Error)
	require.Len(t, records, 4)
	for _, record := range records {
		require.Equal(t, 0, record.UsageCount)
	}
}

func dispatchMIEnvForTest(t *testing.T, action string, payload map[string]any, now time.Time) map[string]any {
	t.Helper()
	cfg := defaultMIEnvCrypto()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	(&MIEnvApi{}).dispatch(ctx, action, payload, cfg, now)
	return decodeMIEnvResponseForTest(t, recorder, cfg, now)
}

func decodeMIEnvResponseForTest(t *testing.T, recorder *httptest.ResponseRecorder, cfg miEnvCryptoConfig, now time.Time) map[string]any {
	t.Helper()
	require.Equal(t, 200, recorder.Code)

	var envelope struct {
		Data string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Greater(t, len(envelope.Data), 6)
	ciphertext, err := decodeMIEnvBase64(envelope.Data[6:])
	require.NoError(t, err)
	var plain []byte
	for _, seed := range miEnvResponseSeeds(cfg, now) {
		plain, err = decryptMIEnvCBC(ciphertext, seed, cfg.IV)
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	require.Greater(t, len(plain), miEnvBlockSize)

	var body map[string]any
	require.NoError(t, json.Unmarshal(plain[miEnvBlockSize:], &body))
	return body
}

func encryptMIEnvRequestForTest(t *testing.T, raw []byte, cfg miEnvCryptoConfig, now time.Time) string {
	t.Helper()
	plain := append(make([]byte, miEnvBlockSize), raw...)
	ciphertext, err := encryptMIEnvCBC(miEnvPKCS7Pad(plain, miEnvBlockSize), miEnvResponseSeed(cfg, now), cfg.IV)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(ciphertext)
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
