package admin

import (
	"goflow/internal/handler"
	"goflow/internal/middleware"
	"goflow/internal/pkg/ratelimit"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(rg *gin.RouterGroup, r *ratelimit.RateLimiter, h *handler.ProductHandler) {
	rg.GET("/products/:id", h.Get)
	// 写接口限流：Redis 异常时 fail-closed
	rg.POST("/products", middleware.RateLimit(r, 1, 3*time.Second, true), h.Create)
	// 读接口限流：Redis 异常时放行（记录告警）
	rg.GET("/products", middleware.RateLimit(r, 1, 1*time.Second, false), h.List)
	rg.PUT("/products/:id", h.Update)
	rg.DELETE("/products/:id", h.Delete)
}
