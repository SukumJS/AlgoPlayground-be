package handlers

import (
	"algoplayground/models"
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetPretestByAlgorithm handles GET /pretests/:algorithm
func GetPretestByAlgorithm(c *gin.Context) {
	algorithm := c.Param("algorithm")

	if algorithm == "" {
		utils.Error(c, http.StatusBadRequest, "algorithm parameter is required")
		return
	}

	result, err := services.GetPretestByAlgorithm(algorithm)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if result == nil {
		utils.Error(c, http.StatusNotFound, "No pretest found for algorithm: "+algorithm)
		return
	}

	utils.Success(c, result)
}

// SubmitPretestAnswers handles POST /pretests/:algorithm/submit
func SubmitPretestAnswers(c *gin.Context) {
	algorithm := c.Param("algorithm")
	uid := c.GetString("uid")

	if algorithm == "" {
		utils.Error(c, http.StatusBadRequest, "algorithm parameter is required")
		return
	}

	if uid == "" {
		utils.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var submission models.PretestSubmission
	if err := c.ShouldBindJSON(&submission); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(submission.Answers) == 0 {
		utils.Error(c, http.StatusBadRequest, "No answers provided")
		return
	}

	result, err := services.GradePretest(uid, algorithm, submission)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, result)
}

// CheckPretestStatus handles GET /pretests/:algorithm/status
func CheckPretestStatus(c *gin.Context) {
	algorithm := c.Param("algorithm")
	uid := c.GetString("uid")

	if algorithm == "" {
		utils.Error(c, http.StatusBadRequest, "algorithm parameter is required")
		return
	}

	if uid == "" {
		utils.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	status, err := services.HasCompletedPretest(uid, algorithm)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, status)
}
