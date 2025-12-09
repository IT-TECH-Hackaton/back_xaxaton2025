package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidFile      = errors.New("неверный формат файла")
	ErrFileTooLarge     = errors.New("файл слишком большой")
	ErrEmptyMessage     = errors.New("сообщение не может быть пустым")
	ErrMessageTooLong   = errors.New("сообщение слишком длинное")
	ErrInvalidSessionID = errors.New("неверный ID сессии")
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(code int, message string, details ...string) *APIError {
	err := &APIError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

func HandleError(c *gin.Context, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		c.JSON(apiErr.Code, apiErr)
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Внутренняя ошибка сервера",
	})
}
