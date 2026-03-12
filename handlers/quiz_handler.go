package handlers

import (
	"algoplayground/models"
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetQuizzes retrieves quizzes by algorithm and typeQuiz
func GetQuizzes(c *gin.Context) {

	algorithm := c.Query("algorithm")
	typeQuiz := c.Query("typeQuiz")

	if algorithm == "" || typeQuiz == "" {
		utils.Error(c, http.StatusBadRequest, "Both 'algorithm' and 'typeQuiz' query parameters are required")
		return
	}

	quizzes, err := services.GetQuizzes(algorithm, typeQuiz)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, quizzes)
}

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
