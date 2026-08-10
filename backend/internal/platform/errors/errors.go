// Package errors 定义领域/业务错误码与错误类型。
package errors

import "fmt"

// Type 错误分类。
type Type string

const (
	TypeProtocol Type = "protocol" // 协议错误：无效 JSON、参数格式
	TypeAuth     Type = "auth"     // 认证授权
	TypeBusiness Type = "business" // 业务错误
	TypeExternal Type = "external" // 外部依赖
	TypeAI       Type = "ai"       // AI 数据
	TypeInternal Type = "internal" // 系统内部
)

// Code 业务码常量。
const (
	CodeSuccess          = "SUCCESS"
	CodeInvalidRequest   = "INVALID_REQUEST"
	CodeInvalidJSON      = "INVALID_JSON"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeUploadTooLarge   = "UPLOAD_TOO_LARGE"

	CodeAuthRequired    = "AUTH_REQUIRED"
	CodeTokenExpired    = "TOKEN_EXPIRED"
	CodeFamilyForbidden = "FAMILY_FORBIDDEN"
	CodeForbidden       = "FORBIDDEN"

	CodeNotFound           = "NOT_FOUND"
	CodeCatNotFound        = "CAT_NOT_FOUND"
	CodeInvalidRecordState = "INVALID_RECORD_STATE"
	CodeConflict           = "CONFLICT"

	CodeAIUnavailable      = "AI_UNAVAILABLE"
	CodeOCRUnavailable     = "OCR_UNAVAILABLE"
	CodeStorageUnavailable = "STORAGE_UNAVAILABLE"

	CodeAIInvalidOutput  = "AI_INVALID_OUTPUT"
	CodeAINeedsClar      = "AI_NEEDS_CLARIFICATION"
	CodeAISessionExpired = "AI_SESSION_EXPIRED"
)

// AppError 统一业务错误。
type AppError struct {
	Type    Type
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

// New 创建业务错误。
func New(t Type, code, message string) *AppError {
	return &AppError{Type: t, Code: code, Message: message}
}

// Wrap 包装底层错误。
func Wrap(t Type, code, message string, err error) *AppError {
	return &AppError{Type: t, Code: code, Message: message, Cause: err}
}

// InvalidRequest 协议/参数错误。
func InvalidRequest(code, message string) *AppError {
	return New(TypeProtocol, code, message)
}

// Unauthorized 认证失败。
func Unauthorized(code, message string) *AppError {
	return New(TypeAuth, code, message)
}

// Forbidden 授权拒绝。
func Forbidden(code, message string) *AppError {
	return New(TypeAuth, code, message)
}

// NotFound 资源不存在（不泄露存在性）。
func NotFound(code, message string) *AppError {
	return New(TypeBusiness, code, message)
}

// Conflict 状态冲突。
func Conflict(code, message string) *AppError {
	return New(TypeBusiness, code, message)
}
