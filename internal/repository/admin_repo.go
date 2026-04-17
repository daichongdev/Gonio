package repository

import (
	"context"

	"gonio/internal/model"

	"gorm.io/gorm"
)

type AdminRepository interface {
	Create(ctx context.Context, admin *model.Admin) error
	GetByUsername(ctx context.Context, username string) (*model.Admin, error)
}

type adminRepo struct {
	db *gorm.DB
}

func NewAdminRepo(db *gorm.DB) AdminRepository {
	return &adminRepo{db: db}
}

// Create 创建管理员
func (r *adminRepo) Create(ctx context.Context, admin *model.Admin) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

// GetByUsername 根据用户名查询管理员
func (r *adminRepo) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}
