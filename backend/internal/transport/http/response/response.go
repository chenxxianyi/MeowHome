// Package response 定义统一响应结构与错误响应映射。
package response

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

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
		appErr = classify(err)
	}

	status := statusFor(appErr)
	c.JSON(status, Envelope{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: RequestID(c),
	})
}

// classify 把非 AppError（主要是 gin 的绑定/校验错误）归类为协议错误，
// 否则请求体格式不对会返回 500，而它实际是客户端问题，应当是 400。
func classify(err error) *errors.AppError {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var validationErrs validator.ValidationErrors
	if stderrors.As(err, &syntaxErr) || stderrors.As(err, &typeErr) ||
		stderrors.As(err, &validationErrs) ||
		stderrors.Is(err, io.EOF) || stderrors.Is(err, io.ErrUnexpectedEOF) {
		return errors.InvalidRequest(errors.CodeInvalidJSON, err.Error())
	}
	return errors.Wrap(errors.TypeInternal, "INTERNAL_ERROR", "internal error", err)
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
		case errors.CodeFamilyForbidden, errors.CodeForbidden:
			// 已认证但无权访问该家庭资源 —— 必须是 403，不能与「未认证」混为一谈
			return http.StatusForbidden
		case errors.CodeAuthRequired, errors.CodeTokenExpired:
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
