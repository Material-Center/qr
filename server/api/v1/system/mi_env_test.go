package system

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
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
