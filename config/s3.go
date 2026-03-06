package config

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var S3Client *s3.Client
var BucketName string

func InitS3() {
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	BucketName = os.Getenv("AWS_BUCKET_NAME")

	if region == "" || accessKey == "" || secretKey == "" || BucketName == "" {
		log.Fatal("Missing AWS S3 environment variables")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)

	if err != nil {
		log.Fatal("Failed to load AWS config: ", err)
	}

	S3Client = s3.NewFromConfig(cfg)
	log.Println("AWS S3 connected")
}
