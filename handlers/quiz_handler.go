package handlers

import (
	"algoplayground/models"
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateQuizzes handles batch creation of quizzes
func CreateQuizzes(c *gin.Context) {
	var quizzes []models.QuizQuestion
	if err := c.ShouldBindJSON(&quizzes); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(quizzes) == 0 {
		utils.Error(c, http.StatusBadRequest, "No quizzes provided")
		return
	}

	if err := services.CreateQuizzes(quizzes); err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, "Quizzes created successfully")
}
