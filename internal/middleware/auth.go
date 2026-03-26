package middleware

import (
	"fmt"
	"strings"

	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthMiddleware struct {
	jwtSecret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: []byte(secret)}
}

func (m *AuthMiddleware) parseToken(c *gin.Context) (*CustomClaims, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Error(c, errcode.ErrUnauthorized())
		c.Abort()
		return nil, false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Error(c, errcode.ErrUnauthorized())
		c.Abort()
		return nil, false
	}

	if len(m.jwtSecret) == 0 {
		response.Error(c, errcode.ErrUnauthorized())
		c.Abort()
		return nil, false
	}

	token, err := jwt.ParseWithClaims(parts[1], &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		response.Error(c, errcode.ErrUnauthorized())
		c.Abort()
		return nil, false
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		response.Error(c, errcode.ErrUnauthorized())
		c.Abort()
		return nil, false
	}

	return claims, true
}

func (m *AuthMiddleware) AppAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := m.parseToken(c)
		if !ok {
			return
		}

		if claims.Role != RoleUser {
			response.Error(c, errcode.ErrForbidden())
			c.Abort()
			return
		}

		c.Set(string(ContextKeyClaims), claims)
		c.Next()
	}
}

func (m *AuthMiddleware) AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := m.parseToken(c)
		if !ok {
			return
		}

		if claims.Role != RoleAdmin {
			response.Error(c, errcode.ErrForbidden())
			c.Abort()
			return
		}

		c.Set(string(ContextKeyClaims), claims)
		c.Next()
	}
}
