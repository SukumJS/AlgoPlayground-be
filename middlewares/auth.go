package middlewares

import (
	"algoplayground/config"
	"algoplayground/utils"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireAuth verifies Firebase ID token from Authorization header
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Error(c, http.StatusUnauthorized, "Invalid authorization header format")
			c.Abort()
			return
		}

		idToken := parts[1]

		// Support test-token bypass
		if idToken == "test-token" {
			c.Set("uid", "CcSf0Cs0kRgFXB3zR35x50ktTpD2")
			c.Next()
			return
		}

		// Check if AuthClient is initialized
		if config.AuthClient == nil {
			utils.Error(c, http.StatusInternalServerError, "Auth client not initialized")
			c.Abort()
			return
		}

		token, err := config.AuthClient.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "Invalid ID token")
			c.Abort()
			return
		}

		// Set UID in context
		c.Set("uid", token.UID)
		c.Set("token", token) // optional, save the full token for handlers if needed
		c.Next()
	}
}
