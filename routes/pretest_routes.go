package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupPretestRoutes(r *gin.Engine) {
	pretests := r.Group("/pretests")
	pretests.Use(middlewares.RequireAuth())

	// GET /pretests/:algorithm — fetch pretest questions (no answers)
	pretests.GET("/:algorithm", handlers.GetPretestByAlgorithm)

	// POST /pretests/:algorithm/submit — submit answers for grading
	pretests.POST("/:algorithm/submit", handlers.SubmitPretestAnswers)

	// GET /pretests/:algorithm/status — check if user completed pretest
	pretests.GET("/:algorithm/status", handlers.CheckPretestStatus)
}
