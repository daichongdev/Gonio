package errcode

import "fmt"

// AppError 统一业务错误类型。
// cause 字段保存原始错误，用于日志记录但不暴露给客户端。
type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	httpStatus int
	cause      error // 原始错误，仅用于内部日志，不序列化到响应
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("code: %d, message: %s, cause: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Is / errors.As 链式解包
func (e *AppError) Unwrap() error {
	return e.cause
}

func (e *AppError) HTTPStatus() int {
	return e.httpStatus
}

// Cause 返回被包装的原始错误，用于日志记录
func (e *AppError) Cause() error {
	return e.cause
}

// Wrap 在 AppError 上附加原始错误，供 service 层使用。
// 原始错误会被记录到日志，但不会暴露给客户端。
func (e *AppError) Wrap(cause error) *AppError {
	e.cause = cause
	return e
}

// New 创建自定义错误
func New(code int, httpStatus int, msg string) *AppError {
	return &AppError{Code: code, httpStatus: httpStatus, Message: msg}
}

// 错误码常量，供外部按 code 值引用（如 response.ErrorWithMsg）
const (
	CodeBadRequest      = 10001
	CodeUnauthorized    = 10002
	CodeForbidden       = 10003
	CodeNotFound        = 10004
	CodeInternal        = 10005
	CodeTooManyRequests = 10006

	CodeUserOrPassword  = 20001
	CodeUserDisabled    = 20002
	CodeAdminOrPassword = 20003
	CodeAdminDisabled   = 20004

	CodeProductNotFound = 20101
	CodeProductOffShelf = 20102
	CodeStockNotEnough  = 20103
)

// 通用错误码 10001-19999
// 每次调用返回新实例，避免全局指针被意外修改

func ErrBadRequest() *AppError {
	return &AppError{Code: CodeBadRequest, Message: "请求参数错误", httpStatus: 400}
}

func ErrUnauthorized() *AppError {
	return &AppError{Code: CodeUnauthorized, Message: "未授权", httpStatus: 401}
}

func ErrForbidden() *AppError {
	return &AppError{Code: CodeForbidden, Message: "禁止访问", httpStatus: 403}
}

func ErrNotFound() *AppError {
	return &AppError{Code: CodeNotFound, Message: "资源不存在", httpStatus: 404}
}

func ErrInternal() *AppError {
	return &AppError{Code: CodeInternal, Message: "服务器内部错误", httpStatus: 500}
}

func ErrTooManyRequests() *AppError {
	return &AppError{Code: CodeTooManyRequests, Message: "请求过于频繁", httpStatus: 429}
}

// 用户错误码 20001-20099

func ErrUserOrPassword() *AppError {
	return &AppError{Code: CodeUserOrPassword, Message: "用户名或密码错误", httpStatus: 401}
}

func ErrUserDisabled() *AppError {
	return &AppError{Code: CodeUserDisabled, Message: "用户已被禁用", httpStatus: 403}
}

func ErrAdminOrPassword() *AppError {
	return &AppError{Code: CodeAdminOrPassword, Message: "管理员账号或密码错误", httpStatus: 401}
}

func ErrAdminDisabled() *AppError {
	return &AppError{Code: CodeAdminDisabled, Message: "管理员账号已被禁用", httpStatus: 403}
}

// 商品错误码 20101-20199

func ErrProductNotFound() *AppError {
	return &AppError{Code: CodeProductNotFound, Message: "商品不存在", httpStatus: 404}
}

func ErrProductOffShelf() *AppError {
	return &AppError{Code: CodeProductOffShelf, Message: "商品已下架", httpStatus: 400}
}

func ErrStockNotEnough() *AppError {
	return &AppError{Code: CodeStockNotEnough, Message: "库存不足", httpStatus: 400}
}
