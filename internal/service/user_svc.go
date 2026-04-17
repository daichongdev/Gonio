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
		return errcode.ErrInternal().Wrap(err)
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
		return nil, errcode.ErrInternal().Wrap(err)
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errcode.ErrUserOrPassword()
	}

	// 检查用户状态
	if user.Status != 1 {
		return nil, errcode.ErrUserDisabled()
	}

	tokenStr, expireAt, err := auth.BuildToken(s.jwtSecret, user.ID, user.Username, auth.RoleUser, s.jwtExpire)
	if err != nil {
		return nil, errcode.ErrInternal().Wrap(err)
	}

	return &response.LoginResult{
		Token:    tokenStr,
		ExpireAt: expireAt,
		User: response.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
	}, nil
}
