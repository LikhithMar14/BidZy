package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Uploader struct {
	Client     *s3.Client
	BucketName string
}

func NewS3Uploader(bucketName *string) (*S3Uploader, error) {
	// Read from your custom environment variable names
	// Handle both the typo and correct spelling
	accessKey := os.Getenv("S3_ACCESS_KEY")
	if accessKey == "" {
		accessKey = os.Getenv("S3_ACESS_KEY") // Fallback to typo version
	}

	secretKey := os.Getenv("S3_SECRET_KEY")
	region := os.Getenv("REGION")

	if accessKey == "" || secretKey == "" || region == "" {
		log.Printf("Missing AWS credentials: accessKey=%v, secretKey=%v, region=%v",
			accessKey != "", secretKey != "", region != "")
		return nil, fmt.Errorf("missing AWS credentials in environment variables")
	}

	log.Printf("AWS Config: region=%s, bucket=%s", region, *bucketName)

	// Create custom credentials provider
	credsProvider := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credsProvider),
	)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Uploader{
		Client:     client,
		BucketName: *bucketName,
	}, nil
}

func (u *S3Uploader) GeneratePresignedPutURL(key string, contentType string) (string, error) {
	presigner := s3.NewPresignClient(u.Client)

	req, err := presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(u.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// GenerateImageURL generates a public S3 URL for an image
func (u *S3Uploader) GenerateImageURL(key string) string {
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", u.BucketName, key)
	log.Printf("Generated S3 URL: %s", url)
	return url
}

// GenerateImageKey generates a unique key for auction images
func (u *S3Uploader) GenerateImageKey(auctionID, fileName string) string {
	ext := ""
	if idx := strings.LastIndex(fileName, "."); idx != -1 {
		ext = fileName[idx:]
	}
	return fmt.Sprintf("auctions/%s/image%s", auctionID, ext)
}

func (u *S3Uploader) HandleGetPresignedURL(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		FileName    string `json:"fileName"`
		ContentType string `json:"contentType"`
	}
	var body Req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("uploads/%d-%s", time.Now().UnixNano(), body.FileName)

	url, err := u.GeneratePresignedPutURL(key, body.ContentType)
	if err != nil {
		http.Error(w, "Failed to generate presigned URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": url,
	})
}
