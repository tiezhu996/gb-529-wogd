package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	HTTPStatus int            `json:"-"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func NewError(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: status}
}

func WithDetails(err *Error, details map[string]any) *Error {
	err.Details = details
	return err
}

var (
	ErrNotFound  = NewError(http.StatusNotFound, "NOT_FOUND", "请求的资源不存在")
	ErrConflict  = NewError(http.StatusConflict, "CONFLICT", "资源状态已变化，请刷新后重试")
	ErrForbidden = NewError(http.StatusForbidden, "ACCESS_DENIED", "当前角色无权执行此操作")
)

func Success(c *gin.Context, status int, data any) {
	response := gin.H{"data": data, "request_id": RequestID(c)}
	c.JSON(status, response)
}

func Page(c *gin.Context, data any, page, pageSize int, total int64) {
	c.JSON(http.StatusOK, gin.H{
		"data":       data,
		"meta":       gin.H{"page": page, "page_size": pageSize, "total": total},
		"request_id": RequestID(c),
	})
}

func Fail(c *gin.Context, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		appErr = NewError(http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时无法完成请求")
	}
	c.AbortWithStatusJSON(appErr.HTTPStatus, gin.H{
		"error":      gin.H{"code": appErr.Code, "message": appErr.Message, "details": appErr.Details},
		"request_id": RequestID(c),
	})
}

func RequestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}
