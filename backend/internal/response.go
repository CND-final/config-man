package app

import (
	"errors"
	"io"
	"net/http"

	"config-man/backend/internal/processor"

	"github.com/gin-gonic/gin"
)

func writeJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

func writeError(c *gin.Context, err error) {
	var appErr *processor.AppError
	if errors.As(err, &appErr) {
		writeJSON(c, appErr.Status, gin.H{"message": appErr.Message})
		return
	}
	writeJSON(c, http.StatusInternalServerError, gin.H{"message": err.Error()})
}

func decodeJSON(c *gin.Context, target any) *processor.AppError {
	if err := c.ShouldBindJSON(target); err != nil {
		return &processor.AppError{Status: http.StatusBadRequest, Message: "Invalid JSON body: " + err.Error()}
	}
	return nil
}

func decodeOptionalJSON(c *gin.Context, target any) *processor.AppError {
	if c.Request.Body == nil {
		return nil
	}
	if err := c.ShouldBindJSON(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return &processor.AppError{Status: http.StatusBadRequest, Message: "Invalid JSON body: " + err.Error()}
	}
	return nil
}
