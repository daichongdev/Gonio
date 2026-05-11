package repository

import (
	"context"

	"gonio/internal/database"
	"gonio/internal/model"

	"gorm.io/gorm"
)

type ProductRepository interface {
	List(ctx context.Context, page, size int) ([]model.Product, int64, error)
	GetByID(ctx context.Context, id uint) (*model.Product, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uint) error
}

type productRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) ProductRepository {
	return &productRepo{db: db}
}

// List 分页查询商品列表
func (r *productRepo) List(ctx context.Context, page, size int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	db := database.GetDB(ctx, r.db)

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
func (r *productRepo) GetByID(ctx context.Context, id uint) (*model.Product, error) {
	var product model.Product
	db := database.GetDB(ctx, r.db)
	if err := db.First(&product, id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// Create 创建商品
func (r *productRepo) Create(ctx context.Context, product *model.Product) error {
	db := database.GetDB(ctx, r.db)
	return db.Create(product).Error
}

// Update 更新商品
func (r *productRepo) Update(ctx context.Context, product *model.Product) error {
	db := database.GetDB(ctx, r.db)
	return db.Save(product).Error
}

// Delete 删除商品（软删除）
func (r *productRepo) Delete(ctx context.Context, id uint) error {
	db := database.GetDB(ctx, r.db)
	result := db.Delete(&model.Product{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
