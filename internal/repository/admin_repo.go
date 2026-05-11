package repository

import (
	"context"

	"gonio/internal/database"
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
	db := database.GetDB(ctx, r.db)
	return db.Create(admin).Error
}

// GetByUsername 根据用户名查询管理员
func (r *adminRepo) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	db := database.GetDB(ctx, r.db)
	if err := db.Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}
