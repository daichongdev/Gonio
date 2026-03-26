package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"goflow/internal/model"
	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/logger"
	"goflow/internal/repository"

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
	rdb         *redis.Client
	cacheExpire time.Duration
}

func NewProductService(repo repository.ProductRepository, rdb *redis.Client, expire int) ProductService {
	if expire <= 0 {
		expire = 600 // 默认 10 分钟
	}
	return &productService{
		repo:        repo,
		rdb:         rdb,
		cacheExpire: time.Duration(expire) * time.Second,
	}
}

func (s *productService) List(ctx context.Context, page, size int) ([]model.Product, int64, error) {
	return s.repo.List(ctx, page, size)
}

func (s *productService) GetByID(ctx context.Context, id uint) (*model.Product, error) {
	cacheKey := fmt.Sprintf("product:%d", id)
	if s.rdb != nil {
		cached, err := s.rdb.Get(ctx, cacheKey).Result()
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
			if s.rdb != nil {
				if setErr := s.rdb.Set(ctx, cacheKey, nullCacheValue, nullCacheTTL).Err(); setErr != nil {
					logger.WithCtx(ctx).Warnw("set null cache failed", "error", setErr)
				}
			}
			return nil, errcode.ErrProductNotFound()
		}
		return nil, errcode.ErrInternal()
	}

	if s.rdb != nil {
		if data, err := json.Marshal(product); err == nil {
			if err := s.rdb.Set(ctx, cacheKey, data, s.cacheExpire).Err(); err != nil {
				logger.WithCtx(ctx).Warnw("set product cache failed", "error", err)
			}
		}
	}

	return product, nil
}

func (s *productService) Create(ctx context.Context, product *model.Product) error {
	return s.repo.Create(ctx, product)
}

func (s *productService) Update(ctx context.Context, product *model.Product) error {
	if err := s.repo.Update(ctx, product); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrProductNotFound()
		}
		return errcode.ErrInternal()
	}
	if s.rdb != nil {
		cacheKey := fmt.Sprintf("product:%d", product.ID)
		if err := s.rdb.Del(ctx, cacheKey).Err(); err != nil {
			logger.WithCtx(ctx).Warnw("delete product cache failed", "error", err)
		}
	}
	return nil
}

func (s *productService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrProductNotFound()
		}
		return errcode.ErrInternal()
	}
	if s.rdb != nil {
		cacheKey := fmt.Sprintf("product:%d", id)
		if err := s.rdb.Del(ctx, cacheKey).Err(); err != nil {
			logger.WithCtx(ctx).Warnw("delete product cache failed", "error", err)
		}
	}
	return nil
}
