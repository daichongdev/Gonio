package middleware

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"math/rand/v2"

	"goflow/internal/pkg/logger"

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

		// 写入 request context，供 logger.WithCtx 使用
		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
