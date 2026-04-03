package handlers

import (
	"algoplayground/models"
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetPosttestByAlgorithm handles GET /posttests/:algorithm
func GetPosttestByAlgorithm(c *gin.Context) {
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

	result, err := services.GetPosttestByAlgorithm(uid, algorithm)
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

// SubmitPosttestAnswers handles POST /posttests/:algorithm/submit
func SubmitPosttestAnswers(c *gin.Context) {
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

	var submission models.PosttestSubmission
	if err := c.ShouldBindJSON(&submission); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(submission.Answers) == 0 {
		utils.Error(c, http.StatusBadRequest, "No answers provided")
		return
	}

	result, err := services.GradePosttest(uid, algorithm, submission)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, result)
}

// SavePosttestProgress handles PUT /posttests/:algorithm/progress
func SavePosttestProgress(c *gin.Context) {
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

	var req models.PosttestProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := services.SavePosttestProgress(uid, algorithm, req.Answers); err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, map[string]bool{"saved": true})
}

// CheckPosttestStatus handles GET /posttests/:algorithm/status
func CheckPosttestStatus(c *gin.Context) {
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

	status, err := services.GetPosttestStatus(uid, algorithm)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, status)
}
