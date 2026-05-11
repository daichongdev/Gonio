package service

import (
	"context"
	"encoding/json"
	"testing"

	"gonio/internal/model"
	"gonio/internal/pkg/cache"
	"gonio/internal/pkg/errcode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// mockProductRepo 是 ProductRepository 的 mock 实现
type mockProductRepo struct {
	mock.Mock
}

func (m *mockProductRepo) List(ctx context.Context, page, size int) ([]model.Product, int64, error) {
	args := m.Called(ctx, page, size)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]model.Product), args.Get(1).(int64), args.Error(2)
}

func (m *mockProductRepo) GetByID(ctx context.Context, id uint) (*model.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *mockProductRepo) Create(ctx context.Context, product *model.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *mockProductRepo) Update(ctx context.Context, product *model.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *mockProductRepo) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// TestProductService_GetByID_CacheHit 测试缓存命中场景
func TestProductService_GetByID_CacheHit(t *testing.T) {
	mockRepo := new(mockProductRepo)
	mockCache := cache.NewMockCache()

	// 预设缓存数据
	product := &model.Product{
		BaseModel: model.BaseModel{ID: 1},
		Name:      "Test Product",
		Price:     99.99,
		Stock:     10,
	}
	data, _ := json.Marshal(product)
	_ = mockCache.Set(context.Background(), "product:1", string(data), 0)

	svc := NewProductService(mockRepo, mockCache, 600)
	result, err := svc.GetByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, "Test Product", result.Name)
	assert.Equal(t, 99.99, result.Price)

	// 验证未访问数据库
	mockRepo.AssertNotCalled(t, "GetByID")
}

// TestProductService_GetByID_CacheMiss 测试缓存未命中场景
func TestProductService_GetByID_CacheMiss(t *testing.T) {
	mockRepo := new(mockProductRepo)
	mockCache := cache.NewMockCache()

	product := &model.Product{
		BaseModel: model.BaseModel{ID: 1},
		Name:      "Test Product",
		Price:     99.99,
		Stock:     10,
	}

	// Mock 数据库返回
	mockRepo.On("GetByID", mock.Anything, uint(1)).Return(product, nil)

	svc := NewProductService(mockRepo, mockCache, 600)
	result, err := svc.GetByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, "Test Product", result.Name)

	// 验证访问了数据库
	mockRepo.AssertCalled(t, "GetByID", mock.Anything, uint(1))

	// 验证缓存已写入
	cached, _ := mockCache.Get(context.Background(), "product:1")
	assert.NotEmpty(t, cached)
}

// TestProductService_GetByID_NotFound 测试商品不存在场景
func TestProductService_GetByID_NotFound(t *testing.T) {
	mockRepo := new(mockProductRepo)
	mockCache := cache.NewMockCache()

	// Mock 数据库返回 NotFound
	mockRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	svc := NewProductService(mockRepo, mockCache, 600)
	result, err := svc.GetByID(context.Background(), 999)

	assert.Nil(t, result)
	assert.Error(t, err)

	// 验证返回的是业务错误码
	var appErr *errcode.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errcode.CodeProductNotFound, appErr.Code)

	// 验证空缓存已写入（防止缓存穿透）
	cached, _ := mockCache.Get(context.Background(), "product:999")
	assert.Equal(t, nullCacheValue, cached)
}

// TestProductService_List 测试列表查询
func TestProductService_List(t *testing.T) {
	mockRepo := new(mockProductRepo)
	mockCache := cache.NewMockCache()

	products := []model.Product{
		{BaseModel: model.BaseModel{ID: 1}, Name: "Product 1", Price: 99.99},
		{BaseModel: model.BaseModel{ID: 2}, Name: "Product 2", Price: 199.99},
	}

	mockRepo.On("List", mock.Anything, 1, 10).Return(products, int64(2), nil)

	svc := NewProductService(mockRepo, mockCache, 600)
	result, total, err := svc.List(context.Background(), 1, 10)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)
	assert.Equal(t, "Product 1", result[0].Name)

	mockRepo.AssertCalled(t, "List", mock.Anything, 1, 10)
}
