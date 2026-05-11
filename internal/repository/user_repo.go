package repository

import (
	"context"

	"gonio/internal/database"
	"gonio/internal/model"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByUsername(ctx context.Context, username string) (*model.User, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

// Create 创建用户
func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	db := database.GetDB(ctx, r.db)
	return db.Create(user).Error
}

// GetByUsername 根据用户名查询用户
func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	db := database.GetDB(ctx, r.db)
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
