package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupExerciseRoutes(r *gin.Engine) {

	exercise := r.Group("/exercise")
	// Use the Auth middleware for all routes in this group
	exercise.Use(middlewares.RequireAuth())

	// Create multiple exercises
	exercise.POST("/batch", handlers.CreateExercises)
}
