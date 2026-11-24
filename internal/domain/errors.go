package domain

import (
	"errors"
	"fmt"
)

// AppError 是对外错误：Code 机器可读，Message 人类可读，Status 映射 HTTP 状态码。
type AppError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func NewError(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func WrapError(code, message string, status int, err error) *AppError {
	return &AppError{Code: code, Message: message, Status: status, Err: err}
}

func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

var (
	ErrNotFound     = NewError("not_found", "资源不存在", 404)
	ErrUnauthorized = NewError("unauthorized", "请先登录", 401)
	ErrForbidden    = NewError("forbidden", "没有权限执行该操作", 403)
	ErrConflict     = NewError("conflict", "资源冲突", 409)
	ErrInvalid      = NewError("validation_error", "请求参数不合法", 400)
	ErrRateLimited  = NewError("rate_limited", "请求过于频繁，请稍后再试", 429)
	ErrInternal     = NewError("internal_error", "服务器内部错误", 500)
)

func NotFound(message string) *AppError {
	return NewError("not_found", message, 404)
}

func Forbidden(code, message string) *AppError {
	return NewError(code, message, 403)
}

func Invalid(code, message string) *AppError {
	return NewError(code, message, 400)
}

func Unauthorized(message string) *AppError {
	return NewError("unauthorized", message, 401)
}

func Conflict(code, message string) *AppError {
	return NewError(code, message, 409)
}
