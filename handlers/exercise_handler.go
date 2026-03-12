package handlers

import (
	"algoplayground/models"
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateExercises handles batch creation of exercises
func CreateExercises(c *gin.Context) {
	var exercises []models.Exercise
	if err := c.ShouldBindJSON(&exercises); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(exercises) == 0 {
		utils.Error(c, http.StatusBadRequest, "No exercises provided")
		return
	}

	if err := services.CreateExercises(exercises); err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, "Exercises created successfully")
}
