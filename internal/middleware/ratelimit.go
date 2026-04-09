package middleware

import (
	"time"

	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/logger"
	"goflow/internal/pkg/ratelimit"
	"goflow/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// RateLimit 基于 IP 和路由的限流中间件
// limit: 窗口内允许的最大请求数
// window: 时间窗口大小
func RateLimit(limiter *ratelimit.RateLimiter, limit int, window time.Duration, failClosed bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}
		ip := c.ClientIP()
		path := c.FullPath()
		if path == "" {
			c.Next()
			return
		}
		key := ratelimit.BuildKey(ip, c.Request.Method, path)
		allowed, retryAfter, err := limiter.Allow(c.Request.Context(), key, limit, window)
		if err != nil {
			logger.WithCtx(c.Request.Context()).Warnw("rate limit check failed", "path", path, "method", c.Request.Method, "ip", ip, "error", err)
			if failClosed {
				response.Error(c, errcode.ErrTooManyRequests())
				c.Abort()
			} else {
				c.Next()
			}
			return
		}

		if !allowed {
			c.Header("Retry-After", ratelimit.RetryAfterSeconds(retryAfter))
			response.Error(c, errcode.ErrTooManyRequests())
			c.Abort()
			return
		}

		c.Next()
	}
}
