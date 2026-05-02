package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupQuizRoutes(r *gin.Engine) {

	quiz := r.Group("/quiz")
	// Use the Auth middleware for all routes in this group
	quiz.Use(middlewares.RequireAuth())

	// Create multiple quizzes
	quiz.POST("/batch", handlers.CreateQuizzes)
}
