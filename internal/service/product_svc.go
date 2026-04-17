package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gonio/internal/model"
	"gonio/internal/pkg/cache"
	"gonio/internal/pkg/errcode"
	"gonio/internal/pkg/logger"
	"gonio/internal/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// nullCacheValue 空缓存标记，防止缓存穿透
const nullCacheValue = "null"

// nullCacheTTL 空缓存 TTL，较短以便数据创建后能快速生效
const nullCacheTTL = 60 * time.Second

type ProductService interface {
	List(ctx context.Context, page, size int) ([]model.Product, int64, error)
	GetByID(ctx context.Context, id uint) (*model.Product, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uint) error
}

type productService struct {
	repo        repository.ProductRepository
	cache       cache.Cache
	cacheExpire time.Duration
}

// NewProductService 创建商品服务。cache 参数可为 nil（跳过缓存，方便测试或无 Redis 部署）。
func NewProductService(repo repository.ProductRepository, c cache.Cache, expire int) ProductService {
	if expire <= 0 {
		expire = 600 // 默认 10 分钟
	}
	return &productService{
		repo:        repo,
		cache:       c,
		cacheExpire: time.Duration(expire) * time.Second,
	}
}

func (s *productService) List(ctx context.Context, page, size int) ([]model.Product, int64, error) {
	return s.repo.List(ctx, page, size)
}

func (s *productService) GetByID(ctx context.Context, id uint) (*model.Product, error) {
	cacheKey := fmt.Sprintf("product:%d", id)
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, cacheKey)
		if err == nil {
			// 空缓存命中：该 ID 不存在，直接返回防止穿透到 DB
			if cached == nullCacheValue {
				return nil, errcode.ErrProductNotFound()
			}
			var product model.Product
			if json.Unmarshal([]byte(cached), &product) == nil {
				return &product, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			logger.WithCtx(ctx).Warnw("get product cache failed", "error", err)
		}
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 缓存空值，防止缓存穿透
			s.setCache(ctx, cacheKey, nullCacheValue, nullCacheTTL)
			return nil, errcode.ErrProductNotFound()
		}
		return nil, errcode.ErrInternal().Wrap(err)
	}

	if data, marshalErr := json.Marshal(product); marshalErr == nil {
		s.setCache(ctx, cacheKey, string(data), s.cacheExpire)
	}

	return product, nil
}

func (s *productService) Create(ctx context.Context, product *model.Product) error {
	if err := s.repo.Create(ctx, product); err != nil {
		return errcode.ErrInternal().Wrap(err)
	}
	s.delCache(ctx, fmt.Sprintf("product:%d", product.ID))
	return nil
}

func (s *productService) Update(ctx context.Context, product *model.Product) error {
	if err := s.repo.Update(ctx, product); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrProductNotFound()
		}
		return errcode.ErrInternal().Wrap(err)
	}
	s.delCache(ctx, fmt.Sprintf("product:%d", product.ID))
	return nil
}

func (s *productService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrProductNotFound()
		}
		return errcode.ErrInternal().Wrap(err)
	}
	s.delCache(ctx, fmt.Sprintf("product:%d", id))
	return nil
}

// setCache 安全地设置缓存，cache 为 nil 时静默跳过
func (s *productService) setCache(ctx context.Context, key, value string, ttl time.Duration) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Set(ctx, key, value, ttl); err != nil {
		logger.WithCtx(ctx).Warnw("set cache failed", "key", key, "error", err)
	}
}

// delCache 安全地删除缓存，cache 为 nil 时静默跳过
func (s *productService) delCache(ctx context.Context, keys ...string) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Del(ctx, keys...); err != nil {
		logger.WithCtx(ctx).Warnw("delete cache failed", "keys", keys, "error", err)
	}
}
