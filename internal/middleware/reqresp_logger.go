package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"gonio/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

// responseWriter 包装 gin.ResponseWriter 以捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// ReqRespLogger 记录请求和响应的详细信息，包括请求体和响应体
func ReqRespLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 判断是否需要记录请求体
		contentType := c.GetHeader("Content-Type")
		shouldLogBody := shouldLogRequestBody(contentType, c.Request.ContentLength)

		// 读取请求体
		var reqBody []byte
		if shouldLogBody && c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			// 恢复请求体供后续 handler 使用
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// 包装 ResponseWriter 以捕获响应体
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = writer

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []interface{}{
			"method", c.Request.Method,
			"path", path,
			"query", query,
			"status", status,
			"latency", latency.String(),
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		// 添加请求体（限制大小避免日志过大）
		if len(reqBody) > 0 {
			maxBodyLog := 2048 // 最多记录 2KB
			if len(reqBody) > maxBodyLog {
				fields = append(fields, "request_body", string(reqBody[:maxBodyLog])+"...(truncated)")
			} else {
				fields = append(fields, "request_body", string(reqBody))
			}
		} else if !shouldLogBody && c.Request.ContentLength > 0 {
			// 跳过的大文件或特殊类型
			fields = append(fields, "request_body", "(skipped: large or binary content)")
		}

		// 添加响应体（限制大小）
		respBody := writer.body.Bytes()
		if len(respBody) > 0 {
			maxBodyLog := 2048
			if len(respBody) > maxBodyLog {
				fields = append(fields, "response_body", string(respBody[:maxBodyLog])+"...(truncated)")
			} else {
				fields = append(fields, "response_body", string(respBody))
			}
		}

		// 如果 handler 通过 c.Error() 注入了错误，一并记录
		if len(c.Errors) > 0 {
			fields = append(fields, "errors", c.Errors.String())
		}

		log := logger.WithCtx(c.Request.Context())

		switch {
		case status >= 500:
			log.Errorw("request_response", fields...)
		case status >= 400:
			log.Warnw("request_response", fields...)
		default:
			log.Infow("request_response", fields...)
		}
	}
}

// shouldLogRequestBody 判断是否应该记录请求体
// 跳过文件上传、二进制流和超大请求体，防止内存溢出
func shouldLogRequestBody(contentType string, contentLength int64) bool {
	// 跳过文件上传
	if strings.Contains(contentType, "multipart/form-data") {
		return false
	}

	// 跳过二进制流
	if strings.Contains(contentType, "application/octet-stream") {
		return false
	}

	// 跳过超过 10KB 的请求体
	if contentLength > 10*1024 {
		return false
	}

	return true
}
