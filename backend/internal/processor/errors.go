package processor

import "net/http"

type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func badRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: message}
}

func unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: message}
}

func forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: message}
}

func notFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: message}
}

func conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: message}
}

func internalError(message string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: message}
}
