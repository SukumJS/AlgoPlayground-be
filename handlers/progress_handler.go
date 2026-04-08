package handlers

import (
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAllProgress handles GET /progress/all
func GetAllProgress(c *gin.Context) {
	uid := c.GetString("uid")

	if uid == "" {
		utils.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	result, err := services.GetAllProgress(uid)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, result)
}
