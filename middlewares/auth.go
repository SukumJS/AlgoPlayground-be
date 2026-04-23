package middlewares

import (
	"algoplayground/config"
	"algoplayground/utils"
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/auth"
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

			c.Set("token", &auth.Token{
				UID: "CcSf0Cs0kRgFXB3zR35x50ktTpD2",
				Claims: map[string]interface{}{
					"email":   "test@gmail.com",
					"picture": "",
					"email_verified": true,
				},
			})

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

		// ดึงค่า email_verified จาก Token Claims มาตรวจสอบ
		// ใช้ type assertion .(bool) เพื่อแปลงค่า
		emailVerified, ok := token.Claims["email_verified"].(bool)
		
		// ถ้าผู้ใช้ล็อกอินด้วย Google ค่านี้จะเป็น true อัตโนมัติ
		// ถ้าสมัครด้วย Email/Password ค่านี้จะเป็น false จนกว่าจะกดลิงก์ในอีเมล
		if !ok || !emailVerified {
			utils.Error(c, http.StatusForbidden, "Please verify your email address first")
			c.Abort()
			return
		}

		// Set UID in context
		c.Set("uid", token.UID)
		c.Set("token", token) // optional, save the full token for handlers if needed
		c.Next()
	}
}