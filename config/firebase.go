package config

import (
	"context"
	"log"
	"os"

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

	opt := option.WithCredentialsFile(cred)

	app, err := firebase.NewApp(ctx, nil, opt)
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
