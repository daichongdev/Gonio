package cache

import (
	"context"
	"time"
)

// Cache 定义缓存操作的最小接口，供 service 层依赖。
// 实现者可以是 Redis、内存缓存、或测试用的 mock。
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}
