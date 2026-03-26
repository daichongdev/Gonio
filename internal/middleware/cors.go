package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件，支持配置允许的来源列表。
// allowedOrigins 为空时允许所有来源（向后兼容开发环境），生产环境应显式配置。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0

	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			// 非跨域请求，跳过 CORS 头写入
			c.Next()
			return
		}

		allowed := allowAll
		if !allowed {
			_, allowed = originSet[origin]
		}

		if !allowed {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID, Accept-Language")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		// Max-Age 单位是秒，浏览器在此时间内缓存预检结果，不再重复发 OPTIONS
		c.Header("Access-Control-Max-Age", "43200") // 12 小时

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
