package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupPosttestRoutes(r *gin.Engine) {
	posttests := r.Group("/posttests")
	posttests.Use(middlewares.RequireAuth())

	// GET /posttests/:algorithm — fetch questions (resumes progress)
	posttests.GET("/:algorithm", handlers.GetPosttestByAlgorithm)

	// POST /posttests/:algorithm/submit — submit answers for grading
	posttests.POST("/:algorithm/submit", handlers.SubmitPosttestAnswers)

	// PUT /posttests/:algorithm/progress — auto-save partial answers
	posttests.PUT("/:algorithm/progress", handlers.SavePosttestProgress)

	// GET /posttests/:algorithm/status — check posttest state
	posttests.GET("/:algorithm/status", handlers.CheckPosttestStatus)
}
