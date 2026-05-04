package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupProgressRoutes(r *gin.Engine) {
	progress := r.Group("/progress")
	progress.Use(middlewares.RequireAuth())

	// GET /progress/all — batch fetch pretest + posttest status for all algorithms
	progress.GET("/all", handlers.GetAllProgress)
}
