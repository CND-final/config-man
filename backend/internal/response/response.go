package response

import (
	"errors"
	"io"

	"config-man/backend/model"

	"github.com/gin-gonic/gin"
)

func WriteJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

func DecodeJSON(c *gin.Context, target any) *model.ErrorDetail {
	if err := c.ShouldBindJSON(target); err != nil {
		return model.InvalidInput("Invalid JSON body: " + err.Error())
	}
	return nil
}

func DecodeOptionalJSON(c *gin.Context, target any) *model.ErrorDetail {
	if c.Request.Body == nil {
		return nil
	}
	if err := c.ShouldBindJSON(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return model.InvalidInput("Invalid JSON body: " + err.Error())
	}
	return nil
}
