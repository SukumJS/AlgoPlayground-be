package services

import (
	"algoplayground/config"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadImageToS3 uploads an image stream to S3 and returns its public URL.
func UploadImageToS3(file io.Reader, filename string, contentType string) (string, error) {
	if config.S3Client == nil {
		return "", fmt.Errorf("S3 client is not initialized")
	}

	_, err := config.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(config.BucketName),
		Key:         aws.String(filename),
		Body:        file,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Format: https://[bucket-name].s3.[region].amazonaws.com/[filename]
	s3Url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", config.BucketName, config.S3Client.Options().Region, filename)
	return s3Url, nil
}

// UploadGinFileToS3 extracts multipart file data (from Gin) and sends it to S3.
func UploadGinFileToS3(fileHeader *multipart.FileHeader, filename string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return UploadImageToS3(file, filename, contentType)
}
