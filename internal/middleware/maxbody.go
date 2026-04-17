package middleware

import (
	"net/http"

	"gonio/internal/pkg/errcode"
	"gonio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 限制请求体大小，防止大请求体攻击导致 OOM。
// maxBytes 为允许的最大字节数，例如 1<<20 表示 1MB。
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()

		// http.MaxBytesReader 在读取超限时会返回特定错误，
		// Gin 的 ShouldBindJSON 会将其转化为 binding error 被 handler 捕获。
		// 此处额外检查 Gin 是否已因此而 abort
		if c.IsAborted() {
			return
		}
		for _, err := range c.Errors {
			if err.Err != nil && err.Err.Error() == "http: request body too large" {
				response.Error(c, errcode.ErrBadRequest())
				c.Abort()
				return
			}
		}
	}
}
