package routes

import (
	"algoplayground/handlers"
	"algoplayground/middlewares"

	"github.com/gin-gonic/gin"
)

func setupAuthRoutes(r *gin.Engine) {

	auth := r.Group("/auth")

	auth.POST("/sync", middlewares.RequireAuth(), handlers.SyncUser)
}
