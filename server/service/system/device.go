package system

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

const (
	deviceHeartbeatKeyPrefix = "device:heartbeat:"
	deviceBusyKeyPrefix      = "device:busy:"
	deviceHeartbeatTTL       = 5 * time.Minute
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
	if err := global.GVA_REDIS.Set(ctx, deviceHeartbeatKey(deviceID), time.Now().Unix(), deviceHeartbeatTTL).Err(); err != nil {
		return err
	}
	return global.GVA_REDIS.Set(ctx, deviceBusyKey(deviceID), value, deviceHeartbeatTTL).Err()
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
	return global.GVA_REDIS.Del(ctx, key).Err()
}

func (s *DeviceService) MarkOffline(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if global.GVA_REDIS == nil {
		return nil
	}
	return global.GVA_REDIS.Del(context.Background(), deviceHeartbeatKey(deviceID), deviceBusyKey(deviceID)).Err()
}

func (s *DeviceService) ListOnlineDeviceIDs() []string {
	return s.listDeviceIDsByPrefix(deviceHeartbeatKeyPrefix)
}

func (s *DeviceService) ListBusyDeviceIDs() []string {
	return s.listDeviceIDsByPrefix(deviceBusyKeyPrefix)
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
