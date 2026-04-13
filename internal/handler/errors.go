package handler

import (
	"errors"

	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func writeServiceError(c *gin.Context, err error) {
	var appErr *errcode.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr)
		return
	}
	response.Error(c, errcode.ErrInternal())
}
