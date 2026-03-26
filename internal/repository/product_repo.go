package repository

import (
	"context"

	"goflow/internal/model"

	"gorm.io/gorm"
)

type ProductRepository interface {
	List(ctx context.Context, page, size int) ([]model.Product, int64, error)
	GetByID(ctx context.Context, id uint) (*model.Product, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uint) error
}

type ProductRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) ProductRepository {
	return &ProductRepo{db: db}
}

// List 分页查询商品列表
func (r *ProductRepo) List(ctx context.Context, page, size int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	db := r.db.WithContext(ctx)

	// 使用独立的查询链，避免 Count 修改 query 内部状态影响后续 Find
	if err := db.Model(&model.Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// total 为 0 时提前返回，省一次 DB 查询
	if total == 0 {
		return products, 0, nil
	}

	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// GetByID 根据 ID 查询商品
func (r *ProductRepo) GetByID(ctx context.Context, id uint) (*model.Product, error) {
	var product model.Product
	if err := r.db.WithContext(ctx).First(&product, id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// Create 创建商品
func (r *ProductRepo) Create(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// Update 更新商品
func (r *ProductRepo) Update(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// Delete 删除商品（软删除）
func (r *ProductRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Product{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
