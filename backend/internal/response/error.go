package response

import (
	"errors"
	"net/http"

	"config-man/backend/model"

	"github.com/gin-gonic/gin"
)

func WriteError(c *gin.Context, err error) {
	var detail *model.ErrorDetail
	if errors.As(err, &detail) {
		WriteJSON(c, StatusFromError(detail), gin.H{
			"kind":    detail.Kind,
			"message": detail.Detail,
		})
		return
	}

	WriteJSON(c, http.StatusInternalServerError, gin.H{
		"kind":    model.ErrorInternal,
		"message": err.Error(),
	})
}

func StatusFromError(err error) int {
	var detail *model.ErrorDetail
	if !errors.As(err, &detail) {
		return http.StatusInternalServerError
	}

	switch detail.Kind {
	case model.ErrorInvalidInput:
		return http.StatusBadRequest
	case model.ErrorUnauthorized:
		return http.StatusUnauthorized
	case model.ErrorForbidden:
		return http.StatusForbidden
	case model.ErrorNotFound:
		return http.StatusNotFound
	case model.ErrorConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
