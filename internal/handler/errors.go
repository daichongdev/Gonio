package handler

import (
	"errors"

	"gonio/internal/pkg/errcode"
	"gonio/internal/pkg/logger"
	"gonio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func writeServiceError(c *gin.Context, err error) {
	var appErr *errcode.AppError
	if errors.As(err, &appErr) {
		// 如果包含原始错误，记录到日志（不暴露给客户端）
		if cause := appErr.Cause(); cause != nil {
			logger.WithCtx(c.Request.Context()).Errorw("service error",
				"code", appErr.Code,
				"cause", cause.Error(),
			)
		}
		response.Error(c, appErr)
		return
	}
	// 未知错误类型，记录完整错误后返回通用 500
	logger.WithCtx(c.Request.Context()).Errorw("unexpected service error", "error", err)
	response.Error(c, errcode.ErrInternal())
}
