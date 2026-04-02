package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupPretestRoutes(r *gin.Engine) {
	pretests := r.Group("/pretests")
	pretests.Use(middlewares.RequireAuth())

	// GET /pretests/:algorithm — fetch pretest questions (resumes progress if exists)
	pretests.GET("/:algorithm", handlers.GetPretestByAlgorithm)

	// POST /pretests/:algorithm/submit — submit answers for grading
	pretests.POST("/:algorithm/submit", handlers.SubmitPretestAnswers)

	// GET /pretests/:algorithm/status — check pretest state (completed/in-progress/not started)
	pretests.GET("/:algorithm/status", handlers.CheckPretestStatus)

	// PUT /pretests/:algorithm/progress — auto-save partial answers
	pretests.PUT("/:algorithm/progress", handlers.SavePretestProgress)
}
