package handler

import (
	"context"
	"fmt"
	"gonio/internal/mq"
	"gonio/internal/pkg/logger"

	"gonio/internal/pkg/errcode"
	"gonio/internal/pkg/i18n"
	"gonio/internal/pkg/req"
	"gonio/internal/pkg/response"
	"gonio/internal/pkg/validator"

	"github.com/gin-gonic/gin"
)

type userLoginService interface {
	Login(ctx context.Context, username, password string) (*response.LoginResult, error)
}

type UserHandler struct {
	userSvc   userLoginService
	publisher *mq.Publisher
}

func NewUserHandler(userSvc userLoginService, publisher *mq.Publisher) *UserHandler {
	return &UserHandler{
		userSvc:   userSvc,
		publisher: publisher,
	}
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var r req.LoginReq
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ErrorWithMsg(c, 400, errcode.CodeBadRequest, validator.Translate(err, i18n.GetLang(c)))
		return
	}

	result, err := h.userSvc.Login(c.Request.Context(), r.Username, r.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	// 发送欢迎邮件
	go func() {
		err = h.publisher.Publish(c, mq.TopicEmail, mq.EmailPayload{
			To:      "daichongdev@gmail.com",
			Subject: "欢迎登录 Gonio",
			Body:    fmt.Sprintf("Hi %s, 欢迎加入！", r.Username),
		})

		if err != nil {
			logger.Log.Warnw("publish welcome email failed", "err", err)
		}
	}()

	response.Success(c, result)
}
