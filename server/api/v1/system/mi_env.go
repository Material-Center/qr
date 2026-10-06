package system

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"github.com/gin-gonic/gin"
)

type MIEnvApi struct{}

func (a *MIEnvApi) Handle(c *gin.Context) {
	action := strings.Trim(strings.TrimSpace(c.Param("action")), "/")
	cfg := defaultMIEnvCrypto()
	now := time.Now()
	if c.Request.Method == http.MethodGet {
		if action != "stats" {
			a.writeFailure(c, cfg, now, "unsupported MI environment GET action")
			return
		}
		a.writeStats(c, cfg, now)
		return
	}
	if c.Request.Method != http.MethodPost {
		c.Header("Allow", http.MethodPost)
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": http.StatusMethodNotAllowed, "msg": "method not allowed"})
		return
	}
	payload, err := a.readPayload(c, cfg, now)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": err.Error()})
		return
	}
	a.dispatch(c, action, payload, cfg, now)
}

func (a *MIEnvApi) dispatch(c *gin.Context, action string, payload map[string]any, cfg miEnvCryptoConfig, now time.Time) {
	svc := system.MIEnvServiceApp
	switch action {
	case "add_env":
		record := model.SysMIEnvRecord{DeviceCode: str(payload["设备代号"]), DeviceID: str(payload["设备ID"]), Type: str(payload["类型"]), SerialBackupName: str(payload["串码备份包名称"]), AndroidID: str(payload["安卓ID"]), Key: str(payload["密钥"]), MaxUsage: 1}
		if missing := firstMissing(map[string]string{"设备代号": record.DeviceCode, "设备ID": record.DeviceID, "类型": record.Type, "串码备份包名称": record.SerialBackupName, "安卓ID": record.AndroidID, "密钥": record.Key}); missing != "" {
			a.writeFailure(c, cfg, now, missing+" is required")
			return
		}
		id, err := svc.Add(record, now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "msg": "添加成功", "success": true, "message": "添加成功", "data": map[string]any{"id": id, "环境id": id}})
	case "get_env", "get_env_enhanced", "get_env_enhanced2":
		record, err := svc.Consume(miEnvFilter(payload, false), now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		if record == nil {
			a.writeFailure(c, cfg, now, "暂无可用环境")
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "msg": "ok", "success": true, "data": miEnvData(record)})
	case "get_env_for_make":
		filter := miEnvFilter(payload, false)
		filter.MaxMadeCount = miEnvIntPtr(payload["已制作次数"])
		filter.CooldownDays = miEnvIntPtr(payload["冷却天数"])
		record, err := svc.ListForMake(filter, now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		if record == nil {
			a.writeFailure(c, cfg, now, "暂无可制作环境")
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "msg": "ok", "success": true, "data": miEnvData(record)})
	case "make_success":
		id, ok := uintValue(payload["环境id"])
		if !ok {
			a.writeFailure(c, cfg, now, "环境id is required")
			return
		}
		record, err := svc.MarkMakeSuccess(id, now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "msg": "制作成功", "success": true, "data": map[string]any{"id": record.ID, "环境id": record.ID, "已制作次数": record.MadeCount}})
	case "increase_make_count":
		a.adjustID(c, cfg, now, payload, func(id uint) error { return svc.AdjustMadeCount(id, 1, false, now) })
	case "decrease_make_count":
		a.adjustID(c, cfg, now, payload, func(id uint) error { return svc.AdjustMadeCount(id, -1, false, now) })
	case "reset_make_count":
		a.adjustID(c, cfg, now, payload, func(id uint) error { return svc.AdjustMadeCount(id, 0, true, now) })
	case "query_env_list":
		a.writeEnvList(c, cfg, now, miEnvFilter(payload, true))
	case "query_env":
		id, ok := uintValue(payload["环境id"])
		if !ok {
			a.writeEnvList(c, cfg, now, miEnvFilter(payload, true))
			return
		}
		record, err := svc.Get(id)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		if record == nil {
			a.writeFailure(c, cfg, now, "环境不存在")
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": miEnvData(record)})
	case "query_by_device":
		deviceID := str(payload["设备ID"])
		if deviceID == "" {
			a.writeFailure(c, cfg, now, "设备ID is required")
			return
		}
		filter := system.MIEnvFilter{DeviceID: deviceID, Limit: miEnvIntPtr(payload["limit"])}
		items, err := svc.List(filter, now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		out := make([]map[string]any, 0, len(items))
		for i := range items {
			out = append(out, miEnvData(&items[i]))
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": out})
	case "freeze_env", "unfreeze_env":
		id, ok := uintValue(payload["环境id"])
		if !ok {
			a.writeFailure(c, cfg, now, "环境id is required")
			return
		}
		err := svc.SetFrozen(id, action == "freeze_env", now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "msg": "ok"})
	case "freeze_by_condition", "unfreeze_by_condition":
		count, err := svc.SetFrozenByFilter(miEnvFilter(payload, true), action == "freeze_by_condition", now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": map[string]any{"影响数量": count}})
	case "delete_env":
		id, ok := uintValue(payload["环境id"])
		if !ok {
			a.writeFailure(c, cfg, now, "环境id is required")
			return
		}
		if err := svc.Delete(id, now); err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "msg": "ok"})
	case "delete_by_condition":
		count, err := svc.DeleteByFilter(miEnvFilter(payload, true), now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": map[string]any{"删除数量": count}})
	case "clean_env":
		count, err := svc.Clean(miEnvIntPtr(payload["超过天数"]), now)
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": map[string]any{"清理数量": count}})
	case "get_env_fixed":
		deviceID := str(payload["设备ID"])
		if deviceID == "" {
			a.writeFailure(c, cfg, now, "设备ID is required")
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": map[string]any{"设备ID": deviceID, "使用本机设备": true, "环境类型": "QQ888", "最大使用次数": 1, "天数限制": 0, "最小制作次数": 1, "最大制作次数": 3, "最小使用次数": 0, "开始日期": "", "结束日期": "", "排序": "创建时间优先"}})
	case "stats_by_type":
		stats, err := svc.StatsByType()
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": stats})
	case "stats_make_progress":
		stats, err := svc.StatsMakeProgress()
		if err != nil {
			a.writeFailure(c, cfg, now, err.Error())
			return
		}
		a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": stats})
	case "total", "available", "frozen", "unused":
		a.writeScalar(c, cfg, now, action)
	default:
		a.writeFailure(c, cfg, now, "unsupported MI environment action: "+action)
	}
}

func (a *MIEnvApi) writeEnvList(c *gin.Context, cfg miEnvCryptoConfig, now time.Time, filter system.MIEnvFilter) {
	items, err := system.MIEnvServiceApp.List(filter, now)
	if err != nil {
		a.writeFailure(c, cfg, now, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, miEnvData(&items[i]))
	}
	a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": out})
}

func (a *MIEnvApi) writeStats(c *gin.Context, cfg miEnvCryptoConfig, now time.Time) {
	stats, err := system.MIEnvServiceApp.Stats()
	if err != nil {
		a.writeFailure(c, cfg, now, err.Error())
		return
	}
	a.writePlain(c, cfg, now, map[string]any{"总数": stats.Total, "可用": stats.Available, "已消费": stats.Consumed, "冻结": stats.Frozen, "已删除": stats.Deleted})
}

func (a *MIEnvApi) writeScalar(c *gin.Context, cfg miEnvCryptoConfig, now time.Time, action string) {
	stats, err := system.MIEnvServiceApp.Stats()
	if err != nil {
		a.writeFailure(c, cfg, now, err.Error())
		return
	}
	value := stats.Total
	name := "总数"
	switch action {
	case "available":
		value, name = stats.Available, "可用"
	case "frozen":
		value, name = stats.Frozen, "冻结"
	case "unused":
		value, name = stats.Unused, "未使用"
	}
	a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "data": map[string]any{name: value}})
}

func (a *MIEnvApi) adjustID(c *gin.Context, cfg miEnvCryptoConfig, now time.Time, payload map[string]any, fn func(uint) error) {
	id, ok := uintValue(payload["环境id"])
	if !ok {
		a.writeFailure(c, cfg, now, "环境id is required")
		return
	}
	if err := fn(id); err != nil {
		a.writeFailure(c, cfg, now, err.Error())
		return
	}
	a.writeSuccess(c, cfg, now, map[string]any{"code": 0, "success": true, "msg": "ok"})
}

func (a *MIEnvApi) readPayload(c *gin.Context, cfg miEnvCryptoConfig, now time.Time) (map[string]any, error) {
	var envelope struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&envelope); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	if envelope.Data == "" {
		return nil, fmt.Errorf("data is required")
	}
	plain, err := decryptMIEnvRequest(envelope.Data, cfg, now)
	if err != nil {
		return nil, fmt.Errorf("decrypt request: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(plain), &payload); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return payload, nil
}

func (a *MIEnvApi) writeFailure(c *gin.Context, cfg miEnvCryptoConfig, now time.Time, message string) {
	a.writePlain(c, cfg, now, map[string]any{"code": 1, "success": false, "msg": message, "message": message})
}
func (a *MIEnvApi) writeSuccess(c *gin.Context, cfg miEnvCryptoConfig, now time.Time, body map[string]any) {
	a.writePlain(c, cfg, now, body)
}
func (a *MIEnvApi) writePlain(c *gin.Context, cfg miEnvCryptoConfig, now time.Time, body map[string]any) {
	raw, err := json.Marshal(body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	encrypted, err := encryptMIEnvResponse(string(raw), cfg, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": encrypted})
}

func miEnvFilter(payload map[string]any, includeList bool) system.MIEnvFilter {
	filter := system.MIEnvFilter{Type: str(payload["类型"]), DeviceCode: str(payload["设备代号"]), DeviceID: str(payload["设备ID"]), SerialBackupName: str(payload["串码备份包名称"]), AndroidID: str(payload["安卓ID"]), Key: str(payload["密钥"]), Frozen: choosePtr(includeList, miEnvIntPtr(payload["冻结"])), Limit: choosePtr(includeList, miEnvIntPtr(payload["limit"])), Offset: choosePtr(includeList, miEnvIntPtr(payload["offset"])), MaxUsage: miEnvIntPtr(payload["最大使用次数"]), MinUsage: miEnvIntPtr(payload["最小使用次数"]), MinMadeCount: miEnvIntPtr(payload["最小制作次数"]), MaxMadeCount: miEnvIntPtr(payload["最大制作次数"]), OlderThanDays: miEnvIntPtr(payload["超过天数"]), MinDays: miEnvIntPtr(payload["最小天数"]), MaxDays: miEnvIntPtr(payload["最大天数"]), CooldownDays: miEnvIntPtr(payload["冷却天数"]), Sort: str(payload["排序"])}
	if madeCount := miEnvIntPtr(payload["已制作次数"]); madeCount != nil {
		filter.MinMadeCount = madeCount
	}
	return filter
}

func miEnvData(record *model.SysMIEnvRecord) map[string]any {
	data := map[string]any{"id": record.ID, "环境id": record.ID, "设备代号": record.DeviceCode, "设备ID": record.DeviceID, "类型": record.Type, "串码备份包名称": record.SerialBackupName, "备份名称": record.SerialBackupName, "安卓ID": record.AndroidID, "密钥": record.Key, "使用次数": record.UsageCount, "最大使用次数": record.MaxUsage, "已制作次数": record.MadeCount, "冻结": boolInt(record.Frozen), "created_at": record.CreatedAt.Format(time.RFC3339), "创建时间": record.CreatedAt.Unix()}
	if record.ConsumedAt != nil {
		data["consumed_at"] = record.ConsumedAt.Format(time.RFC3339)
	}
	if record.DeletedAt != nil {
		data["deleted_at"] = record.DeletedAt.Format(time.RFC3339)
	}
	if record.LastUsedAt != nil {
		data["最后使用时间"] = record.LastUsedAt.Unix()
	}
	if record.MakeReservedUntil != nil {
		data["制作预约到期时间"] = record.MakeReservedUntil.Format(time.RFC3339)
	}
	return data
}

func str(value any) string {
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}
func miEnvIntPtr(value any) *int {
	switch v := value.(type) {
	case float64:
		n := int(v)
		return &n
	case int:
		return &v
	case json.Number:
		n, _ := strconv.Atoi(string(v))
		return &n
	}
	return nil
}
func uintValue(value any) (uint, bool) {
	if p := miEnvIntPtr(value); p != nil && *p > 0 {
		return uint(*p), true
	}
	return 0, false
}
func choosePtr(enabled bool, value *int) *int {
	if enabled {
		return value
	}
	return nil
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func firstMissing(values map[string]string) string {
	for _, key := range []string{"设备代号", "设备ID", "类型", "串码备份包名称", "安卓ID", "密钥"} {
		if values[key] == "" {
			return key
		}
	}
	return ""
}
