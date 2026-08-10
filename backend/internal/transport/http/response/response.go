// Package response 定义统一响应结构与错误响应映射。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/meowhome/backend/internal/platform/errors"
)

// Envelope 统一响应包裹。
type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// Meta 分页元信息。
type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// PageData 分页数据包裹。
type PageData struct {
	Items any  `json:"items"`
	Meta  Meta `json:"meta"`
}

// OK 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:      errors.CodeSuccess,
		Message:   "ok",
		Data:      data,
		RequestID: RequestID(c),
	})
}

// Created 创建成功。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{
		Code:      errors.CodeSuccess,
		Message:   "created",
		Data:      data,
		RequestID: RequestID(c),
	})
}

// NoContent 无内容。
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Err 业务错误响应。
func Err(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		appErr = errors.Wrap(errors.TypeInternal, "INTERNAL_ERROR", "internal error", err)
	}

	status := statusFor(appErr)
	c.JSON(status, Envelope{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: RequestID(c),
	})
}

// statusFor 将错误类别映射为 HTTP 状态码。
func statusFor(e *errors.AppError) int {
	switch e.Type {
	case errors.TypeProtocol:
		if e.Code == errors.CodeUploadTooLarge {
			return http.StatusRequestEntityTooLarge
		}
		return http.StatusBadRequest
	case errors.TypeAuth:
		switch e.Code {
		case errors.CodeAuthRequired:
			return http.StatusUnauthorized
		case errors.CodeTokenExpired:
			return http.StatusUnauthorized
		default:
			return http.StatusUnauthorized
		}
	case errors.TypeBusiness:
		switch e.Code {
		case errors.CodeNotFound, errors.CodeCatNotFound:
			return http.StatusNotFound
		case errors.CodeConflict, errors.CodeInvalidRecordState:
			return http.StatusConflict
		default:
			return http.StatusBadRequest
		}
	case errors.TypeExternal:
		return http.StatusServiceUnavailable
	case errors.TypeAI:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// RequestID 从 gin context 提取请求 ID。
func RequestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
