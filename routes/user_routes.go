package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupUserRoutes(r *gin.Engine) {
	user := r.Group("/user")
	user.Use(middlewares.RequireAuth())

	// Profile Picture Management
	user.PUT("/profile-image", handlers.UpdateProfileImage)
}
