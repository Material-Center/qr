package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEnvAddAndGetConsumesOnce(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Store:  store,
		Now:    func() time.Time { return now },
	})

	addResp := postEncryptedEnv(t, srv, cfg, now, "/add_env", map[string]any{
		"设备代号":    "cepheus",
		"设备ID":    "8bf9321c",
		"类型":      "QQ888",
		"串码备份包名称": "backup-a",
		"安卓ID":    "android-a",
		"密钥":      "key-a",
	})
	if addResp["success"] != true {
		t.Fatalf("add response = %#v", addResp)
	}

	getResp := postEncryptedEnv(t, srv, cfg, now, "/get_env", map[string]any{
		"类型":     "QQ888",
		"设备代号":   "cepheus",
		"设备ID":   "8bf9321c",
		"最大使用次数": 0,
	})
	data, ok := getResp["data"].(map[string]any)
	if !ok {
		t.Fatalf("get data = %#v in %#v", getResp["data"], getResp)
	}
	if data["备份名称"] != "backup-a" || data["安卓ID"] != "android-a" || data["密钥"] != "key-a" {
		t.Fatalf("get data = %#v", data)
	}

	secondResp := postEncryptedEnv(t, srv, cfg, now, "/get_env", map[string]any{
		"类型":     "QQ888",
		"设备代号":   "cepheus",
		"设备ID":   "8bf9321c",
		"最大使用次数": 0,
	})
	if secondResp["success"] != false {
		t.Fatalf("second get response = %#v, want unavailable", secondResp)
	}
}

func TestEnvUploadIsIdempotentAndUsageLimitComesFromRequest(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	srv := NewServer(ServerConfig{Crypto: cfg, Store: store, Now: func() time.Time { return now }})
	payload := map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
		"串码备份包名称": "same-backup", "安卓ID": "android-a", "密钥": "key-a",
	}
	first := postEncryptedEnv(t, srv, cfg, now, "/add_env", payload)
	second := postEncryptedEnv(t, srv, cfg, now, "/add_env", payload)
	if first["data"].(map[string]any)["环境id"] != second["data"].(map[string]any)["环境id"] {
		t.Fatalf("duplicate upload ids differ: %#v %#v", first, second)
	}

	filter := map[string]any{"类型": "QQ888", "设备ID": "device-a", "最大使用次数": 1}
	for wantUsage := 1; wantUsage <= 2; wantUsage++ {
		resp := postEncryptedEnv(t, srv, cfg, now, "/get_env", filter)
		if resp["success"] != true || resp["data"].(map[string]any)["使用次数"] != float64(wantUsage) {
			t.Fatalf("usage %d response = %#v", wantUsage, resp)
		}
	}
	third := postEncryptedEnv(t, srv, cfg, now, "/get_env", filter)
	if third["success"] != false {
		t.Fatalf("third response = %#v", third)
	}
}

func TestEnvEnhanced2AppliesDateRangesAndSortOrder(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	for _, item := range []struct {
		name string
		age  int
	}{{"older", 20}, {"newer", 10}} {
		created := now.Add(-time.Duration(item.age) * 24 * time.Hour)
		srv := NewServer(ServerConfig{Crypto: cfg, Store: store, Now: func() time.Time { return created }})
		postEncryptedEnv(t, srv, cfg, created, "/add_env", map[string]any{
			"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
			"串码备份包名称": item.name, "安卓ID": "android-" + item.name, "密钥": "key-" + item.name,
		})
	}
	srv := NewServer(ServerConfig{Crypto: cfg, Store: store, Now: func() time.Time { return now }})
	resp := postEncryptedEnv(t, srv, cfg, now, "/get_env_enhanced2", map[string]any{
		"类型": "QQ888", "设备ID": "device-a", "最小使用次数": 0, "最大使用次数": 1,
		"最小天数": 30, "最大天数": 7, "最小制作次数": 1, "最大制作次数": 3,
		"排序": "创建时间优先",
	})
	if resp["data"].(map[string]any)["串码备份包名称"] != "older" {
		t.Fatalf("enhanced2 response = %#v", resp)
	}
}

func TestEnvGetRespectsOlderThanDays(t *testing.T) {
	cfg := DefaultEnvConfig()
	createdAt := fixedLATime()
	store := newTestStore(t)
	addSrv := NewServer(ServerConfig{
		Crypto: cfg,
		Store:  store,
		Now:    func() time.Time { return createdAt },
	})
	postEncryptedEnv(t, addSrv, cfg, createdAt, "/add_env", map[string]any{
		"设备代号":    "cepheus",
		"设备ID":    "8bf9321c",
		"类型":      "QQ888",
		"串码备份包名称": "backup-a",
		"安卓ID":    "android-a",
		"密钥":      "key-a",
	})

	tooEarly := createdAt.Add(48 * time.Hour)
	earlySrv := NewServer(ServerConfig{
		Crypto: cfg,
		Store:  store,
		Now:    func() time.Time { return tooEarly },
	})
	earlyResp := postEncryptedEnv(t, earlySrv, cfg, tooEarly, "/get_env", map[string]any{
		"类型":   "QQ888",
		"设备代号": "cepheus",
		"设备ID": "8bf9321c",
		"超过天数": 3,
	})
	if earlyResp["success"] != false {
		t.Fatalf("early get response = %#v, want unavailable", earlyResp)
	}

	readyAt := createdAt.Add(72 * time.Hour)
	readySrv := NewServer(ServerConfig{
		Crypto: cfg,
		Store:  store,
		Now:    func() time.Time { return readyAt },
	})
	readyResp := postEncryptedEnv(t, readySrv, cfg, readyAt, "/get_env", map[string]any{
		"类型":   "QQ888",
		"设备代号": "cepheus",
		"设备ID": "8bf9321c",
		"超过天数": 3,
	})
	if readyResp["success"] != true {
		t.Fatalf("ready get response = %#v, want available", readyResp)
	}
}

func TestEnvQueryListDoesNotConsume(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Store:  store,
		Now:    func() time.Time { return now },
	})
	postEncryptedEnv(t, srv, cfg, now, "/add_env", map[string]any{
		"设备代号":    "cepheus",
		"设备ID":    "8bf9321c",
		"类型":      "QQ888",
		"串码备份包名称": "backup-a",
		"安卓ID":    "android-a",
		"密钥":      "key-a",
	})

	queryResp := postEncryptedEnv(t, srv, cfg, now, "/query_env_list", map[string]any{
		"类型":   "QQ888",
		"设备代号": "cepheus",
	})
	items, ok := queryResp["data"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("query data = %#v in %#v", queryResp["data"], queryResp)
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("query item = %#v", items[0])
	}
	if _, ok := item["创建时间"].(float64); !ok {
		t.Fatalf("创建时间 must be a Unix-second JSON number: %#v", item["创建时间"])
	}
	if got, ok := item["最后使用时间"].(float64); !ok || got != 0 {
		t.Fatalf("最后使用时间 must use the never-used sentinel 0: %#v", item["最后使用时间"])
	}

	getResp := postEncryptedEnv(t, srv, cfg, now, "/get_env", map[string]any{
		"类型":   "QQ888",
		"设备代号": "cepheus",
	})
	if getResp["success"] != true {
		t.Fatalf("get after query response = %#v", getResp)
	}
}

func TestEnvFreezeDeleteAndStats(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	srv := NewServer(ServerConfig{
		Crypto: cfg,
		Store:  store,
		Now:    func() time.Time { return now },
	})
	addResp := postEncryptedEnv(t, srv, cfg, now, "/add_env", map[string]any{
		"设备代号":    "cepheus",
		"设备ID":    "8bf9321c",
		"类型":      "QQ888",
		"串码备份包名称": "backup-a",
		"安卓ID":    "android-a",
		"密钥":      "key-a",
	})
	id := int(addResp["data"].(map[string]any)["环境id"].(float64))

	postEncryptedEnv(t, srv, cfg, now, "/freeze_env", map[string]any{"环境id": id})
	frozenResp := postEncryptedEnv(t, srv, cfg, now, "/get_env", map[string]any{"类型": "QQ888"})
	if frozenResp["success"] != false {
		t.Fatalf("frozen get response = %#v", frozenResp)
	}

	postEncryptedEnv(t, srv, cfg, now, "/unfreeze_env", map[string]any{"环境id": id})
	availableResp := postEncryptedEnv(t, srv, cfg, now, "/get_env", map[string]any{"类型": "QQ888"})
	if availableResp["success"] != true {
		t.Fatalf("unfrozen get response = %#v", availableResp)
	}

	addResp = postEncryptedEnv(t, srv, cfg, now, "/add_env", map[string]any{
		"设备代号":    "cepheus",
		"设备ID":    "8bf9321c",
		"类型":      "QQ888",
		"串码备份包名称": "backup-b",
		"安卓ID":    "android-b",
		"密钥":      "key-b",
	})
	id = int(addResp["data"].(map[string]any)["环境id"].(float64))
	postEncryptedEnv(t, srv, cfg, now, "/delete_env", map[string]any{"环境id": id})

	stats := getPlainJSON(t, srv, "/stats")
	if stats["总数"] != float64(2) || stats["已消费"] != float64(1) || stats["已删除"] != float64(1) {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestLatestEnvMakeLifecycleAndFixedConfig(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	srv := NewServer(ServerConfig{Crypto: cfg, Store: store, Now: func() time.Time { return now }})
	add := postEncryptedEnv(t, srv, cfg, now, "/add_env", map[string]any{
		"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
		"串码备份包名称": "backup-a", "安卓ID": "android-a", "密钥": "key-a",
	})
	id := int(add["data"].(map[string]any)["环境id"].(float64))

	forMake := postEncryptedEnv(t, srv, cfg, now, "/get_env_for_make", map[string]any{
		"类型": "QQ888", "设备代号": "cepheus", "设备ID": "device-a", "已制作次数": 3, "冷却天数": 1,
	})
	data := forMake["data"].(map[string]any)
	if data["已制作次数"] != float64(1) {
		t.Fatalf("for make data = %#v", data)
	}
	postEncryptedEnv(t, srv, cfg, now, "/make_success", map[string]any{"环境id": id})
	query := postEncryptedEnv(t, srv, cfg, now, "/query_env", map[string]any{"环境id": id})
	if query["data"].(map[string]any)["已制作次数"] != float64(2) {
		t.Fatalf("query = %#v", query)
	}

	fixed := postEncryptedEnv(t, srv, cfg, now, "/get_env_fixed", map[string]any{"设备ID": "device-a"})
	fixedData := fixed["data"].(map[string]any)
	if fixedData["环境类型"] != "QQ888" || fixedData["排序"] != "创建时间优先" {
		t.Fatalf("fixed = %#v", fixed)
	}
}

func TestLatestEnvConditionManagement(t *testing.T) {
	cfg := DefaultEnvConfig()
	now := fixedLATime()
	store := newTestStore(t)
	srv := NewServer(ServerConfig{Crypto: cfg, Store: store, Now: func() time.Time { return now }})
	for _, name := range []string{"backup-a", "backup-b"} {
		postEncryptedEnv(t, srv, cfg, now, "/add_env", map[string]any{
			"设备代号": "cepheus", "设备ID": "device-a", "类型": "QQ888",
			"串码备份包名称": name, "安卓ID": "android-" + name, "密钥": "key-" + name,
		})
	}
	frozen := postEncryptedEnv(t, srv, cfg, now, "/freeze_by_condition", map[string]any{"类型": "QQ888", "设备ID": "device-a"})
	if frozen["data"].(map[string]any)["影响数量"] != float64(2) {
		t.Fatalf("freeze = %#v", frozen)
	}
	count := postEncryptedEnv(t, srv, cfg, now, "/frozen", map[string]any{})
	if count["data"].(map[string]any)["冻结"] != float64(2) {
		t.Fatalf("frozen count = %#v", count)
	}
	deleted := postEncryptedEnv(t, srv, cfg, now, "/delete_by_condition", map[string]any{"串码备份包名称": "backup-a"})
	if deleted["data"].(map[string]any)["删除数量"] != float64(1) {
		t.Fatalf("delete = %#v", deleted)
	}
}

func postEncryptedEnv(t *testing.T, srv *Server, cfg CryptoConfig, now time.Time, path string, payload map[string]any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal env payload: %v", err)
	}
	encrypted := encryptEnvRequestFixture(t, string(raw), cfg, now)
	body, err := json.Marshal(map[string]string{"data": encrypted})
	if err != nil {
		t.Fatalf("marshal env envelope: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	srv.EnvHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d, body = %s", path, rec.Code, rec.Body.String())
	}

	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope["code"] != float64(0) {
		t.Fatalf("response envelope code = %v, want 0", envelope["code"])
	}
	encryptedResp, ok := envelope["data"].(string)
	if !ok || encryptedResp == "" {
		t.Fatalf("response envelope = %#v", envelope)
	}
	plain, err := decryptResponseStringAt(encryptedResp, cfg, now)
	if err != nil {
		t.Fatalf("decrypt env response: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(plain), &resp); err != nil {
		t.Fatalf("decode env response plaintext %q: %v", plain, err)
	}
	return resp
}

func encryptEnvRequestFixture(t *testing.T, plain string, cfg CryptoConfig, now time.Time) string {
	t.Helper()

	prefix := make([]byte, aesBlockSize)
	if _, err := rand.Read(prefix); err != nil {
		t.Fatalf("read prefix: %v", err)
	}
	padded := pkcs7Pad(append(prefix, []byte(plain)...), aesBlockSize)
	encrypted, err := encryptCBC(padded, responseSeed(cfg, now), cfg.IV)
	if err != nil {
		t.Fatalf("encrypt env request: %v", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted)
}

func getPlainJSON(t *testing.T, srv *Server, path string) map[string]any {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	srv.EnvHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d, body = %s", path, rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode plain json: %v", err)
	}
	return resp
}
