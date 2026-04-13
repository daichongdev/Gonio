package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

func BuildToken(secret []byte, userID uint, username string, role Role, expire time.Duration) (string, int64, error) {
	expireAt := time.Now().Add(expire)
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		return "", 0, err
	}
	return tokenStr, expireAt.Unix(), nil
}
