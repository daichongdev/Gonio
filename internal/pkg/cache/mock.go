package cache

import (
	"context"
	"time"
)

// MockCache 用于测试的 Mock Cache 实现
type MockCache struct {
	data map[string]string
}

// NewMockCache 创建 Mock Cache 实例
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]string),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", ErrCacheMiss
}

func (m *MockCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCache) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *MockCache) GetOrLoad(ctx context.Context, key string, loader func() (string, error), ttl time.Duration) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	val, err := loader()
	if err != nil {
		return "", err
	}
	m.data[key] = val
	return val, nil
}

// ErrCacheMiss 缓存未命中错误
var ErrCacheMiss = &cacheError{msg: "cache miss"}

type cacheError struct {
	msg string
}

func (e *cacheError) Error() string {
	return e.msg
}
