package middleware

import (
	"encoding/binary"
	"encoding/hex"
	"math/rand/v2"

	"gonio/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"

// generateRequestID 生成轻量级请求 ID，使用 math/rand 避免 crypto/rand 的系统调用开销
func generateRequestID() string {
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], rand.Uint64())
	binary.LittleEndian.PutUint64(buf[8:], rand.Uint64())
	return hex.EncodeToString(buf[:])
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = generateRequestID()
		}
		c.Set(string(logger.RequestIDKey), reqID)
		c.Header(RequestIDHeader, reqID)

		// 使用 logger.NewContext 一次性创建带 request_id 的 logger 并缓存到 context，
		// 后续 logger.WithCtx 直接复用，避免每次调用都分配新的 SugaredLogger。
		ctx := logger.NewContext(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
