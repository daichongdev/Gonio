package service

import (
	"context"
	"errors"
	"time"

	"gonio/internal/config"
	"gonio/internal/model"
	"gonio/internal/pkg/auth"
	"gonio/internal/pkg/errcode"
	"gonio/internal/pkg/response"
	"gonio/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminService interface {
	CreateAdmin(ctx context.Context, username, password, nickname, role string) error
	Login(ctx context.Context, username, password string) (*response.LoginResult, error)
}

type adminService struct {
	repo      repository.AdminRepository
	jwtSecret []byte
	jwtExpire time.Duration
}

func NewAdminService(repo repository.AdminRepository, cfg *config.Config) AdminService {
	jwtExpire := cfg.JWT.Expire
	if jwtExpire <= 0 {
		jwtExpire = 7200
	}
	return &adminService{
		repo:      repo,
		jwtSecret: []byte(cfg.JWT.Secret),
		jwtExpire: time.Duration(jwtExpire) * time.Second,
	}
}

// CreateAdmin 创建管理员
func (s *adminService) CreateAdmin(ctx context.Context, username, password, nickname, role string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errcode.ErrInternal().Wrap(err)
	}
	return s.repo.Create(ctx, &model.Admin{
		Username: username,
		Password: string(hashed),
		Nickname: nickname,
		Role:     role,
		Status:   1,
	})
}

// Login 管理员登录
func (s *adminService) Login(ctx context.Context, username, password string) (*response.LoginResult, error) {
	admin, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdminOrPassword()
		}
		return nil, errcode.ErrInternal().Wrap(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, errcode.ErrAdminOrPassword()
	}

	if admin.Status != 1 {
		return nil, errcode.ErrAdminDisabled()
	}

	tokenStr, expireAt, err := auth.BuildToken(s.jwtSecret, admin.ID, admin.Username, auth.RoleAdmin, s.jwtExpire)
	if err != nil {
		return nil, errcode.ErrInternal().Wrap(err)
	}

	return &response.LoginResult{
		Token:    tokenStr,
		ExpireAt: expireAt,
		User: response.AdminInfo{
			ID:       admin.ID,
			Username: admin.Username,
			Nickname: admin.Nickname,
			Avatar:   admin.Avatar,
			Role:     admin.Role,
		},
	}, nil
}
