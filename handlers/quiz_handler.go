package handlers

import (
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
