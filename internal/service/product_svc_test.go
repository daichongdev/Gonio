package service

import (
	"context"
	"testing"
	"time"

	"gonio/internal/model"
	"gonio/internal/pkg/errcode"

	"gorm.io/gorm"
)

// ── mock repository ──

type mockProductRepo struct {
	product   *model.Product
	products  []model.Product
	total     int64
	err       error
	deleteErr error
}

func (m *mockProductRepo) List(_ context.Context, _, _ int) ([]model.Product, int64, error) {
	return m.products, m.total, m.err
}

func (m *mockProductRepo) GetByID(_ context.Context, _ uint) (*model.Product, error) {
	return m.product, m.err
}

func (m *mockProductRepo) Create(_ context.Context, p *model.Product) error {
	p.ID = 1 // simulate auto-increment
	return m.err
}

func (m *mockProductRepo) Update(_ context.Context, _ *model.Product) error {
	return m.err
}

func (m *mockProductRepo) Delete(_ context.Context, _ uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

// ── mock cache ──

type mockCache struct {
	store map[string]string
	err   error
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	v, ok := m.store[key]
	if !ok {
		// 模拟 redis.Nil 行为 — 返回一个普通 error
		return "", errNotFound
	}
	return v, nil
}

func (m *mockCache) Set(_ context.Context, key string, value string, _ time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.store[key] = value
	return nil
}

func (m *mockCache) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.store, k)
	}
	return nil
}

// errNotFound 模拟 key 不存在（非 redis.Nil，但也不是异常错误）
var errNotFound = gorm.ErrRecordNotFound // 复用一个已知的 sentinel

// ── tests ──

func TestProductService_GetByID_FromDB(t *testing.T) {
	product := &model.Product{Name: "test", Price: 999}
	product.ID = 42

	repo := &mockProductRepo{product: product}
	cache := newMockCache()
	svc := NewProductService(repo, cache, 600)

	got, err := svc.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "test" {
		t.Fatalf("expected name 'test', got '%s'", got.Name)
	}
	// 验证缓存已写入
	if _, ok := cache.store["product:42"]; !ok {
		t.Fatal("expected cache to be populated after DB read")
	}
}

func TestProductService_GetByID_FromCache(t *testing.T) {
	cache := newMockCache()
	cache.store["product:1"] = `{"id":1,"name":"cached","price":100}`

	// repo 返回错误 — 如果走到 DB 说明缓存没命中
	repo := &mockProductRepo{err: gorm.ErrRecordNotFound}
	svc := NewProductService(repo, cache, 600)

	got, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "cached" {
		t.Fatalf("expected name 'cached', got '%s'", got.Name)
	}
}

func TestProductService_GetByID_NullCache(t *testing.T) {
	cache := newMockCache()
	cache.store["product:999"] = nullCacheValue

	repo := &mockProductRepo{}
	svc := NewProductService(repo, cache, 600)

	_, err := svc.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for null-cached product")
	}
	appErr, ok := err.(*errcode.AppError)
	if !ok {
		t.Fatalf("expected *errcode.AppError, got %T", err)
	}
	if appErr.Code != errcode.CodeProductNotFound {
		t.Fatalf("expected code %d, got %d", errcode.CodeProductNotFound, appErr.Code)
	}
}

func TestProductService_GetByID_NotFound_SetsNullCache(t *testing.T) {
	cache := newMockCache()
	repo := &mockProductRepo{err: gorm.ErrRecordNotFound}
	svc := NewProductService(repo, cache, 600)

	_, err := svc.GetByID(context.Background(), 404)
	if err == nil {
		t.Fatal("expected error")
	}
	// 验证空缓存已写入
	v, ok := cache.store["product:404"]
	if !ok || v != nullCacheValue {
		t.Fatal("expected null cache to be set for missing product")
	}
}

func TestProductService_Create_ClearsCache(t *testing.T) {
	cache := newMockCache()
	cache.store["product:1"] = "stale"

	repo := &mockProductRepo{}
	svc := NewProductService(repo, cache, 600)

	p := &model.Product{Name: "new"}
	if err := svc.Create(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cache.store["product:1"]; ok {
		t.Fatal("expected cache to be cleared after create")
	}
}

func TestProductService_Delete_NotFound(t *testing.T) {
	repo := &mockProductRepo{deleteErr: gorm.ErrRecordNotFound}
	svc := NewProductService(repo, nil, 600)

	err := svc.Delete(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr, ok := err.(*errcode.AppError)
	if !ok {
		t.Fatalf("expected *errcode.AppError, got %T", err)
	}
	if appErr.Code != errcode.CodeProductNotFound {
		t.Fatalf("expected code %d, got %d", errcode.CodeProductNotFound, appErr.Code)
	}
}

func TestProductService_NilCache_StillWorks(t *testing.T) {
	product := &model.Product{Name: "no-cache"}
	product.ID = 10
	repo := &mockProductRepo{product: product}

	// cache = nil — 应当正常工作，只是跳过缓存
	svc := NewProductService(repo, nil, 600)

	got, err := svc.GetByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "no-cache" {
		t.Fatalf("expected name 'no-cache', got '%s'", got.Name)
	}
}
