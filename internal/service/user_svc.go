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

type UserService interface {
	CreateUser(ctx context.Context, username, password, nickname string) error
	Login(ctx context.Context, username, password string) (*response.LoginResult, error)
}

type userService struct {
	repo      repository.UserRepository
	jwtSecret []byte
	jwtExpire time.Duration
}

func NewUserService(repo repository.UserRepository, cfg *config.Config) UserService {
	jwtExpire := cfg.JWT.Expire
	if jwtExpire <= 0 {
		jwtExpire = 7200
	}
	return &userService{
		repo:      repo,
		jwtSecret: []byte(cfg.JWT.Secret),
		jwtExpire: time.Duration(jwtExpire) * time.Second,
	}
}

// CreateUser 创建用户
func (s *userService) CreateUser(ctx context.Context, username, password, nickname string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errcode.ErrInternal()
	}
	return s.repo.Create(ctx, &model.User{
		Username: username,
		Password: string(hashed),
		Nickname: nickname,
		Status:   1,
	})
}

func (s *userService) Login(ctx context.Context, username, password string) (*response.LoginResult, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrUserOrPassword()
		}
		return nil, errcode.ErrInternal()
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errcode.ErrUserOrPassword()
	}

	// 检查用户状态
	if user.Status != 1 {
		return nil, errcode.ErrUserDisabled()
	}

	// 生成 JWT — 使用 CustomClaims 与认证中间件保持一致
	expireAt := time.Now().Add(s.jwtExpire)
	claims := middleware.CustomClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     middleware.RoleUser,
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
		User: response.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
	}, nil
}
