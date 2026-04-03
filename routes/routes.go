package routes

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {

	setupAuthRoutes(r)
	setupQuizRoutes(r)
	setupPretestRoutes(r)
	setupUserRoutes(r)
	setupExerciseRoutes(r)
	setupPosttestRoutes(r)
}
