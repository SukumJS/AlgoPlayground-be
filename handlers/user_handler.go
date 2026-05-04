package handlers

import (
	"algoplayground/services"
	"algoplayground/utils"
	"net/http"

	"firebase.google.com/go/auth"
	"github.com/gin-gonic/gin"
)

func SyncUser(c *gin.Context) {
	// Get validated Firebase token from Auth middleware
	tokenInterface, exists := c.Get("token")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Token not found in context")
		return
	}

	token, ok := tokenInterface.(*auth.Token)
	if !ok {
		utils.Error(c, http.StatusInternalServerError, "Invalid token type in context")
		return
	}

	syncResp, err := services.SyncUser(token)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message": "user synced successfully",
		"user":    syncResp,
	})
}

func UpdateProfileImage(c *gin.Context) {
	// Get token from middleware context
	tokenInterface, exists := c.Get("token")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "Token not found in context")
		return
	}
	token, ok := tokenInterface.(*auth.Token)
	if !ok {
		utils.Error(c, http.StatusInternalServerError, "Invalid token type")
		return
	}

	uid := token.UID

	// Parse multipart form
	fileHeader, err := c.FormFile("profileImage")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "Image file is required")
		return
	}

	// Verify image format
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		utils.Error(c, http.StatusBadRequest, "Invalid file format. Only JPEG, PNG, and WebP are allowed")
		return
	}

	url, err := services.UpdateUserProfileImage(uid, fileHeader)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message":  "profile image updated successfully",
		"imageUrl": url,
	})
}
