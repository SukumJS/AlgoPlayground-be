package services

import (
	"algoplayground/config"
	"algoplayground/models"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SyncUser(token *auth.Token) (*models.User, error) {
	ctx := context.Background()

	email, _ := token.Claims["email"].(string)
	picture, _ := token.Claims["picture"].(string)
	name, _ := token.Claims["name"].(string)

	// Fallback name
	if name == "" && email != "" {
		name = strings.Split(email, "@")[0]
	}

	docRef := config.Firestore.Collection("users").Doc(token.UID)
	doc, err := docRef.Get(ctx)

	var user models.User

	if status.Code(err) == codes.NotFound {
		// If Google picture is missing, provide a default UI-Avatar.
		if picture == "" {
			defaultName := "User"
			if name != "" {
				defaultName = url.QueryEscape(name) // Fallback to token name
			}
			uiAvatarUrl := fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=random", defaultName)

			// Save to S3 permanently.
			s3Url, err := uploadURLToS3(uiAvatarUrl, fmt.Sprintf("profiles/%s.jpg", token.UID))
			if err == nil {
				picture = s3Url // Override with our own S3 hosted link
			} else {
				picture = uiAvatarUrl // Fallback
			}
		} else {
			// Mirror Google image to S3
			s3Url, err := uploadURLToS3(picture, fmt.Sprintf("profiles/%s.jpg", token.UID))
			if err == nil {
				picture = s3Url
			}
		}

		// Create user in Firestore
		user = models.User{
			ID:        token.UID,
			ImageURL:  picture,
			UpdatedAt: time.Now(),
		}
		if _, err = docRef.Set(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check user: %v", err)
	} else {
		// Load existing
		doc.DataTo(&user)
	}

	return &user, nil
}

// uploadURLToS3 downloads an image and pipes it directly to S3
func uploadURLToS3(imageUrl, filename string) (string, error) {
	resp, err := http.Get(imageUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return UploadImageToS3(resp.Body, filename, contentType)
}

// UpdateUserProfileImage uploads a new profile picture to S3 and updates Firestore
func UpdateUserProfileImage(uid string, fileHeader *multipart.FileHeader) (string, error) {
	ctx := context.Background()
	filename := fmt.Sprintf("profiles/%s.jpg", uid) // We overwrite the same file

	s3Url, err := UploadGinFileToS3(fileHeader, filename)
	if err != nil {
		return "", fmt.Errorf("failed to upload new profile to S3: %v", err)
	}

	// Appending timestamp to strictly bypass browser cache
	uniqueUrl := fmt.Sprintf("%s?t=%d", s3Url, time.Now().Unix())

	docRef := config.Firestore.Collection("users").Doc(uid)
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "imageUrl", Value: uniqueUrl},
		{Path: "updatedAt", Value: time.Now()},
	})

	if err != nil {
		return "", fmt.Errorf("failed to update user doc in firestore: %v", err)
	}

	return uniqueUrl, nil
}
