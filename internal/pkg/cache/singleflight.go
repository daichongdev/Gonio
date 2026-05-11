package cache

import (
	"context"
	"time"

	"golang.org/x/sync/singleflight"
)

// CacheWithSingleflight 带防击穿保护的缓存包装器
type CacheWithSingleflight struct {
	cache Cache
	sf    singleflight.Group
}

// NewCacheWithSingleflight 创建带 singleflight 保护的缓存实例
func NewCacheWithSingleflight(cache Cache) *CacheWithSingleflight {
	return &CacheWithSingleflight{
		cache: cache,
	}
}

// GetOrLoad 获取缓存，如果缓存未命中则调用 loader 加载数据。
// 使用 singleflight 确保同一时刻只有一个请求执行 loader，其他请求等待结果。
// 这可以防止缓存击穿（大量并发请求同时击穿到数据库）。
//
// 使用示例：
//
//	val, err := c.GetOrLoad(ctx, "product:123", func() (string, error) {
//	    product, err := repo.GetByID(ctx, 123)
//	    if err != nil {
//	        return "", err
//	    }
//	    data, _ := json.Marshal(product)
//	    return string(data), nil
//	}, 10*time.Minute)
func (c *CacheWithSingleflight) GetOrLoad(ctx context.Context, key string, loader func() (string, error), ttl time.Duration) (string, error) {
	// 先查缓存
	val, err := c.cache.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	// 缓存未命中，使用 singleflight 防止击穿
	result, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// 再次检查缓存（可能其他协程已加载）
		if val, err := c.cache.Get(ctx, key); err == nil {
			return val, nil
		}

		// 加载数据
		val, err := loader()
		if err != nil {
			return nil, err
		}

		// 写入缓存（忽略写入错误，不影响返回结果）
		_ = c.cache.Set(ctx, key, val, ttl)
		return val, nil
	})

	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// Get 直接获取缓存（不使用 singleflight）
func (c *CacheWithSingleflight) Get(ctx context.Context, key string) (string, error) {
	return c.cache.Get(ctx, key)
}

// Set 设置缓存
func (c *CacheWithSingleflight) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.cache.Set(ctx, key, value, ttl)
}

// Del 删除缓存
func (c *CacheWithSingleflight) Del(ctx context.Context, keys ...string) error {
	return c.cache.Del(ctx, keys...)
}
