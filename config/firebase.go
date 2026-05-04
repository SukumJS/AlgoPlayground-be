package config

import (
	"context"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"google.golang.org/api/option"
)

var Firestore *firestore.Client
var AuthClient *auth.Client

func InitFirebase() {
	ctx := context.Background()

	cred := os.Getenv("FIREBASE_CREDENTIAL")
	if cred == "" {
		log.Fatal("FIREBASE_CREDENTIAL not set in env")
	}

	var opt option.ClientOption

	if strings.HasPrefix(strings.TrimSpace(cred), "{") {
		opt = option.WithCredentialsJSON([]byte(cred))
		log.Println("Using credentials from JSON")
	} else {
		opt = option.WithCredentialsFile(cred)
		log.Println("Using credentials from File")
	}

	// 1. ดึง Project ID มาจาก Environment 
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		// ถ้าลืมใส่ใน .env หรือลืมใส่ใน Render ให้มันแจ้งเตือน
		log.Println("WARNING: FIREBASE_PROJECT_ID is empty!")
	}

	// 2. ยัด Project ID ใส่ Config 
	config := &firebase.Config{
		ProjectID: projectID,
	}

	// 3. เปลี่ยนตรงนี้จาก nil ให้เป็น config แทน 
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Fatalf("firebase init error: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("firestore init error: %v", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("firebase auth init error: %v", err)
	}

	Firestore = client
	AuthClient = authClient

	log.Println("Firebase connected")
}