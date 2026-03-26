package service

import (
	"context"
	"errors"
	"time"

	"goflow/internal/config"
	"goflow/internal/middleware"
	"goflow/internal/model"
	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/response"
	"goflow/internal/repository"

	"github.com/golang-jwt/jwt/v5"
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
		return errcode.ErrInternal()
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
		return nil, errcode.ErrInternal()
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, errcode.ErrAdminOrPassword()
	}

	if admin.Status != 1 {
		return nil, errcode.ErrAdminDisabled()
	}

	expireAt := time.Now().Add(s.jwtExpire)
	claims := middleware.CustomClaims{
		UserID:   admin.ID,
		Username: admin.Username,
		Role:     middleware.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errcode.ErrInternal()
	}

	return &response.LoginResult{
		Token:    tokenStr,
		ExpireAt: expireAt.Unix(),
		User: response.AdminInfo{
			ID:       admin.ID,
			Username: admin.Username,
			Nickname: admin.Nickname,
			Avatar:   admin.Avatar,
			Role:     admin.Role,
		},
	}, nil
}
