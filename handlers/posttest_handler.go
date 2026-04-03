package handlers

import (
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetPosttestByAlgorithm handles GET /posttests/:algorithm
func GetPosttestByAlgorithm(c *gin.Context) {
	algorithm := c.Param("algorithm")

	if algorithm == "" {
		utils.Error(c, http.StatusBadRequest, "algorithm parameter is required")
		return
	}

	result, err := services.GetPosttestByAlgorithm(algorithm)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if result == nil {
		utils.Error(c, http.StatusNotFound, "No posttest found for algorithm: "+algorithm)
		return
	}

	utils.Success(c, result)
}
