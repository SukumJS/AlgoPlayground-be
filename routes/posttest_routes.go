package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupPosttestRoutes(r *gin.Engine) {
	posttests := r.Group("/posttests")
	posttests.Use(middlewares.RequireAuth())

	// GET /posttests/:algorithm — fetch posttest questions (random 5, ≥1 per type)
	posttests.GET("/:algorithm", handlers.GetPosttestByAlgorithm)
}
