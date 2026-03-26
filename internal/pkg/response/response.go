package response

import (
	"fmt"
	"goflow/internal/pkg/errcode"
	"goflow/internal/pkg/i18n"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMsg 成功响应（自定义消息）
func SuccessWithMsg(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: msg,
		Data:    data,
	})
}

// Error 错误响应（使用 AppError，自动根据语言翻译）
func Error(c *gin.Context, err *errcode.AppError) {
	lang := i18n.GetLang(c)
	msg := i18n.GetMessage(err.Code, lang)
	if msg == "" {
		msg = err.Message
	}

	// 将错误注入 Gin 错误链，让 Logger 中间件能够捕获并记录
	_ = c.Error(fmt.Errorf("[%d] %s", err.Code, msg))

	c.JSON(err.HTTPStatus(), Response{
		Code:    err.Code,
		Message: msg,
	})
}

// ErrorWithMsg 错误响应（自定义）
func ErrorWithMsg(c *gin.Context, httpStatus int, code int, msg string) {
	// 将错误注入 Gin 错误链，让 Logger 中间件能够捕获并记录
	_ = c.Error(fmt.Errorf("[%d] %s", code, msg))

	c.JSON(httpStatus, Response{
		Code:    code,
		Message: msg,
	})
}

// PageData 分页数据
type PageData struct {
	List  any   `json:"list"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, list any, total int64, page, size int) {
	Success(c, PageData{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
