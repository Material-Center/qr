package utils

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/redis/go-redis/v9"
)

const loginUserEnableCacheTTL = 5 * time.Minute

func GetLoginUserEnableCache(ctx context.Context, userUUID string) (int, bool, error) {
	userUUID = strings.TrimSpace(userUUID)
	if userUUID == "" || global.GVA_REDIS == nil {
		return 0, false, nil
	}
	raw, err := global.GVA_REDIS.Get(ctx, loginUserEnableCacheKey(userUUID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	enable, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false, err
	}
	return enable, true, nil
}

func SetLoginUserEnableCache(ctx context.Context, userUUID string, enable int) error {
	userUUID = strings.TrimSpace(userUUID)
	if userUUID == "" || global.GVA_REDIS == nil {
		return nil
	}
	return global.GVA_REDIS.Set(ctx, loginUserEnableCacheKey(userUUID), strconv.Itoa(enable), loginUserEnableCacheTTL).Err()
}

func loginUserEnableCacheKey(userUUID string) string {
	return fmt.Sprintf("login:user:enable:%s", strings.TrimSpace(userUUID))
}
