package system

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

const (
	deviceHeartbeatKeyPrefix = "device:heartbeat:"
	deviceBusyKeyPrefix      = "device:busy:"
	deviceCooldownKeyPrefix  = "device:cooldown:"
	deviceHeartbeatTTL       = 5 * time.Minute
	deviceHeartbeatFreshness = 30 * time.Second
)

type DeviceService struct{}

func (s *DeviceService) MarkHeartbeat(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if global.GVA_REDIS == nil {
		return nil
	}
	return global.GVA_REDIS.Set(context.Background(), deviceHeartbeatKey(deviceID), time.Now().Unix(), deviceHeartbeatTTL).Err()
}

func (s *DeviceService) MarkBusy(deviceID string, business string) error {
	return s.MarkBusyWithTTL(deviceID, business, deviceHeartbeatTTL)
}

func (s *DeviceService) MarkCooldown(deviceID string, ttl time.Duration) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if global.GVA_REDIS == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = deviceHeartbeatTTL
	}
	if err := global.GVA_REDIS.Set(context.Background(), deviceCooldownKey(deviceID), time.Now().Unix(), ttl).Err(); err != nil {
		return err
	}
	resetPhoneRegisterDeviceStatsCache()
	return nil
}

func (s *DeviceService) MarkBusyWithTTL(deviceID string, business string, ttl time.Duration) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if global.GVA_REDIS == nil {
		return nil
	}
	ctx := context.Background()
	value := strings.TrimSpace(business)
	if value == "" {
		value = "busy"
	}
	if ttl <= 0 {
		ttl = deviceHeartbeatTTL
	}
	if err := global.GVA_REDIS.Set(ctx, deviceHeartbeatKey(deviceID), time.Now().Unix(), deviceHeartbeatTTL).Err(); err != nil {
		return err
	}
	if err := global.GVA_REDIS.Set(ctx, deviceBusyKey(deviceID), value, ttl).Err(); err != nil {
		return err
	}
	resetPhoneRegisterDeviceStatsCache()
	return nil
}

func (s *DeviceService) TryReserveIdleDevice(business string, ttl time.Duration) (string, error) {
	if global.GVA_REDIS == nil {
		return "", nil
	}
	value := strings.TrimSpace(business)
	if value == "" {
		value = "busy"
	}
	if ttl <= 0 {
		ttl = deviceHeartbeatTTL
	}
	ctx := context.Background()
	for _, deviceID := range s.ListOnlineDeviceIDs() {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		ok, err := global.GVA_REDIS.SetNX(ctx, deviceBusyKey(deviceID), value, ttl).Result()
		if err != nil {
			return "", err
		}
		if ok {
			resetPhoneRegisterDeviceStatsCache()
			return deviceID, nil
		}
	}
	return "", nil
}

func (s *DeviceService) UpdateBusyIfMatching(deviceID string, oldBusiness string, newBusiness string, ttl time.Duration) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || global.GVA_REDIS == nil {
		return nil
	}
	oldValue := strings.TrimSpace(oldBusiness)
	newValue := strings.TrimSpace(newBusiness)
	if oldValue == "" || newValue == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = deviceHeartbeatTTL
	}
	ctx := context.Background()
	key := deviceBusyKey(deviceID)
	current, err := global.GVA_REDIS.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	if strings.TrimSpace(current) != oldValue {
		return fmt.Errorf("设备%s busy状态已变更", deviceID)
	}
	return global.GVA_REDIS.Set(ctx, key, newValue, ttl).Err()
}

func (s *DeviceService) ClearBusy(deviceID string, businesses ...string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if global.GVA_REDIS == nil {
		return nil
	}
	ctx := context.Background()
	key := deviceBusyKey(deviceID)
	if len(businesses) > 0 && strings.TrimSpace(businesses[0]) != "" {
		current, err := global.GVA_REDIS.Get(ctx, key).Result()
		if err != nil {
			return nil
		}
		if strings.TrimSpace(current) != strings.TrimSpace(businesses[0]) {
			return nil
		}
	}
	if err := global.GVA_REDIS.Del(ctx, key).Err(); err != nil {
		return err
	}
	resetPhoneRegisterDeviceStatsCache()
	return nil
}

func (s *DeviceService) MarkOffline(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if global.GVA_REDIS == nil {
		return nil
	}
	if err := global.GVA_REDIS.Del(context.Background(), deviceHeartbeatKey(deviceID), deviceBusyKey(deviceID)).Err(); err != nil {
		return err
	}
	resetPhoneRegisterDeviceStatsCache()
	return nil
}

func (s *DeviceService) ListOnlineDeviceIDs() []string {
	devices := s.listDeviceIDsByPrefix(deviceHeartbeatKeyPrefix)
	if len(devices) == 0 || global.GVA_REDIS == nil {
		return devices
	}
	now := time.Now()
	keys := make([]string, 0, len(devices))
	for _, deviceID := range devices {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID != "" {
			keys = append(keys, deviceHeartbeatKey(deviceID))
		}
	}
	if len(keys) == 0 {
		return nil
	}
	values, err := global.GVA_REDIS.MGet(context.Background(), keys...).Result()
	if err != nil {
		return nil
	}
	freshDevices := make([]string, 0, len(devices))
	for index, deviceID := range devices {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		if index >= len(values) || values[index] == nil {
			continue
		}
		heartbeatUnix, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(values[index])), 10, 64)
		if err != nil {
			continue
		}
		heartbeatAt := time.Unix(heartbeatUnix, 0)
		if heartbeatAt.After(now.Add(time.Second)) {
			continue
		}
		if now.Sub(heartbeatAt) <= deviceHeartbeatFreshness {
			freshDevices = append(freshDevices, deviceID)
		}
	}
	return freshDevices
}

func (s *DeviceService) ListBusyDeviceIDs() []string {
	return s.listDeviceIDsByPrefix(deviceBusyKeyPrefix)
}

func (s *DeviceService) ListCooldownDeviceIDs() []string {
	return s.listDeviceIDsByPrefix(deviceCooldownKeyPrefix)
}

func (s *DeviceService) listDeviceIDsByPrefix(prefix string) []string {
	if global.GVA_REDIS == nil {
		return nil
	}
	ctx := context.Background()
	pattern := prefix + "*"
	var cursor uint64
	var devices []string
	for {
		keys, nextCursor, err := global.GVA_REDIS.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			if global.GVA_LOG != nil {
				global.GVA_LOG.Warn("设备心跳读取失败", zap.Error(err))
			}
			return devices
		}
		for _, key := range keys {
			deviceID := strings.TrimSpace(strings.TrimPrefix(key, prefix))
			if deviceID != "" && deviceID != key {
				devices = append(devices, deviceID)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return devices
}

func deviceHeartbeatKey(deviceID string) string {
	return deviceHeartbeatKeyPrefix + strings.TrimSpace(deviceID)
}

func deviceBusyKey(deviceID string) string {
	return deviceBusyKeyPrefix + strings.TrimSpace(deviceID)
}

func deviceCooldownKey(deviceID string) string {
	return deviceCooldownKeyPrefix + strings.TrimSpace(deviceID)
}
