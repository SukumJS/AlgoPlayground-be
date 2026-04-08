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

// SyncResponse wraps the Firestore user with email from Firebase token (not stored in Firestore)
type SyncResponse struct {
	models.User
	Email string `json:"email"`
}

func defaultUserProgress(uid string) models.UserProgress {
	return models.UserProgress{
		UserID:        uid,
		TotalProgress: 0,
		PretestScore:  0,
		PosttestScore: 0,
	}
}

func defaultUserCategoryAlgoProgress(uid string) models.UserCategoryAlgoProgressInProfile {
	return models.UserCategoryAlgoProgressInProfile{
		UserID:    uid,
		Linear:    0,
		Trees:     0,
		Graph:     0,
		Sorting:   0,
		Searching: 0,
	}
}

func ensureUserProfileDefaults(ctx context.Context, docRef *firestore.DocumentRef, doc *firestore.DocumentSnapshot, user *models.User, uid string) error {
	updates := make([]firestore.Update, 0, 2)

	if _, err := doc.DataAt("progress"); err != nil || user.Progress.UserID == "" {
		user.Progress = defaultUserProgress(uid)
		updates = append(updates, firestore.Update{Path: "progress", Value: user.Progress})
	}

	if _, err := doc.DataAt("categoryAlgoProgress"); err != nil || user.CategoryAlgoProgress.UserID == "" {
		user.CategoryAlgoProgress = defaultUserCategoryAlgoProgress(uid)
		updates = append(updates, firestore.Update{Path: "categoryAlgoProgress", Value: user.CategoryAlgoProgress})
	}

	if len(updates) == 0 {
		return nil
	}

	if _, err := docRef.Update(ctx, updates); err != nil {
		return fmt.Errorf("failed to initialize user profile progress fields: %v", err)
	}

	return nil
}

func SyncUser(token *auth.Token) (*SyncResponse, error) {
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

		// Create user in Firestore (email intentionally NOT stored — read from Firebase Auth token)
		user = models.User{
			ID:                   token.UID,
			ImageURL:             picture,
			UpdatedAt:            time.Now(),
			Progress:             defaultUserProgress(token.UID),
			CategoryAlgoProgress: defaultUserCategoryAlgoProgress(token.UID),
		}
		if _, err = docRef.Set(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check user: %v", err)
	} else {
		// Load existing
		doc.DataTo(&user)

		if err := ensureUserProfileDefaults(ctx, docRef, doc, &user, token.UID); err != nil {
			return nil, err
		}
	}

	return &SyncResponse{User: user, Email: email}, nil
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
