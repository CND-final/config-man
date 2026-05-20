package apperror

import "net/http"

type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func BadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: message}
}

func Internal(message string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: message}
}
