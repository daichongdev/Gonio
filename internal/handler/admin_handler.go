package handler

import (
	"context"
	"errors"

	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/i18n"
	"goflow/internal/pkg/req"
	"goflow/internal/pkg/response"
	"goflow/internal/pkg/validator"

	"github.com/gin-gonic/gin"
)

type adminLoginService interface {
	Login(ctx context.Context, username, password string) (*response.LoginResult, error)
}

type AdminHandler struct {
	adminSvc adminLoginService
}

func NewAdminHandler(adminSvc adminLoginService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc}
}

// Login 管理员登录
func (h *AdminHandler) Login(c *gin.Context) {
	var r req.AdminLoginReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ErrorWithMsg(c, 400, errcode.CodeBadRequest, validator.Translate(err, i18n.GetLang(c)))
		return
	}

	result, err := h.adminSvc.Login(c.Request.Context(), r.Username, r.Password)
	if err != nil {
		var appErr *errcode.AppError
		if errors.As(err, &appErr) {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errcode.ErrInternal())
		return
	}

	response.Success(c, result)
}
