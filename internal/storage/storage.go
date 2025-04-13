package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/google/uuid"
)

// StorageService is an interface for different storage implementations
type StorageService interface {
	// UploadFaceImage uploads a face image and returns the file URL
	UploadFaceImage(ctx context.Context, file *multipart.FileHeader, personHash string, organizationID string) (string, error)
	
	// DeleteFaceImage deletes a face image from storage
	DeleteFaceImage(ctx context.Context, fileURL string) error
}

// LocalStorageService implements StorageService for local filesystem storage
type LocalStorageService struct {
	StoragePath string
	BaseURL     string
}

// NewLocalStorageService creates a new local storage service
func NewLocalStorageService(storagePath, baseURL string) (*LocalStorageService, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("unable to create storage directory: %w", err)
	}
	return &LocalStorageService{
		StoragePath: storagePath,
		BaseURL:     baseURL,
	}, nil
}

// UploadFaceImage implements StorageService interface for local storage
func (s *LocalStorageService) UploadFaceImage(ctx context.Context, file *multipart.FileHeader, personHash string, organizationID string) (string, error) {
	// Create organization directory
	orgPath := filepath.Join(s.StoragePath, organizationID)
	if err := os.MkdirAll(orgPath, 0755); err != nil {
		return "", fmt.Errorf("unable to create organization directory: %w", err)
	}

	// Generate a unique filename
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s_%s%s", personHash, uuid.New().String(), fileExt)
	filePath := filepath.Join(orgPath, fileName)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("unable to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err = io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("unable to copy file: %w", err)
	}

	// Return the URL to access the file
	fileURL := fmt.Sprintf("%s/%s/%s", s.BaseURL, organizationID, fileName)
	return fileURL, nil
}

// DeleteFaceImage implements StorageService interface for local storage
func (s *LocalStorageService) DeleteFaceImage(ctx context.Context, fileURL string) error {
	// Extract file path from URL
	// This assumes the fileURL format is: baseURL/organizationID/fileName
	relativePath := fileURL[len(s.BaseURL)+1:]
	filePath := filepath.Join(s.StoragePath, relativePath)

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			// File already doesn't exist, not an error
			return nil
		}
		return fmt.Errorf("unable to delete file: %w", err)
	}

	return nil
}

// S3StorageService implements StorageService for S3 or compatible storage
type S3StorageService struct {
	Client    *s3.Client
	BucketName string
	BaseURL   string
}

// NewS3StorageService creates a new S3 storage service
func NewS3StorageService(cfg *appconfig.Config) (*S3StorageService, error) {
	// Create custom endpoint resolver if provided
	var endpointResolver aws.EndpointResolverWithOptions
	if cfg.S3Endpoint != "" {
		endpointResolver = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.S3Endpoint,
				SigningRegion:     cfg.S3Region,
				HostnameImmutable: true,
			}, nil
		})
	}

	// Configure credentials
	credProvider := credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")

	// Configure S3 client
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credProvider),
		awsconfig.WithEndpointResolverWithOptions(endpointResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to configure S3 client: %w", err)
	}

	// Create S3 client with optional path-style addressing
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3UsePathStyle {
			o.UsePathStyle = true
		}
	})

	// Check if bucket exists, if not create it
	_, err = s3Client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(cfg.S3Bucket),
	})
	if err != nil {
		// Bucket doesn't exist, create it
		_, err = s3Client.CreateBucket(context.Background(), &s3.CreateBucketInput{
			Bucket: aws.String(cfg.S3Bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("unable to create S3 bucket: %w", err)
		}
	}

	return &S3StorageService{
		Client:     s3Client,
		BucketName: cfg.S3Bucket,
		BaseURL:    fmt.Sprintf("%s/%s", cfg.S3Endpoint, cfg.S3Bucket),
	}, nil
}

// UploadFaceImage implements StorageService interface for S3 storage
func (s *S3StorageService) UploadFaceImage(ctx context.Context, file *multipart.FileHeader, personHash string, organizationID string) (string, error) {
	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("unable to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate object key
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s/%s_%s%s", organizationID, personHash, uuid.New().String(), fileExt)

	// Upload to S3
	_, err = s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.BucketName),
		Key:         aws.String(fileName),
		Body:        src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", fmt.Errorf("unable to upload file to S3: %w", err)
	}

	// Return the URL to access the file
	fileURL := fmt.Sprintf("%s/%s", s.BaseURL, fileName)
	return fileURL, nil
}

// DeleteFaceImage implements StorageService interface for S3 storage
func (s *S3StorageService) DeleteFaceImage(ctx context.Context, fileURL string) error {
	// Extract object key from URL
	// This assumes the fileURL format is: baseURL/objectKey
	objectKey := fileURL[len(s.BaseURL)+1:]

	// Delete from S3
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("unable to delete file from S3: %w", err)
	}

	return nil
}

// NewStorageService creates a storage service based on configuration
func NewStorageService(cfg *appconfig.Config) (StorageService, error) {
	if cfg.S3Enabled {
		return NewS3StorageService(cfg)
	}
	
	// Default to local storage
	storagePath := "./storage/faces"
	baseURL := fmt.Sprintf("http://localhost:%s/api/faces", cfg.Port)
	return NewLocalStorageService(storagePath, baseURL)
}