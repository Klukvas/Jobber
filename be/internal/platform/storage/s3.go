package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client provides S3 storage operations
type S3Client struct {
	client *s3.Client
	bucket string
}

// NewS3Client creates a new S3 client with custom endpoint support
func NewS3Client(cfg config.S3Config) (*S3Client, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("S3 configuration is incomplete")
	}

	// Create custom resolver for Hetzner endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// Create AWS config with custom credentials and endpoint
	awsConfig := aws.Config{
		Region:                      cfg.Region,
		Credentials:                 credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		EndpointResolverWithOptions: customResolver,
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true // Required for S3-compatible storage
	})

	return &S3Client{
		client: s3Client,
		bucket: cfg.Bucket,
	}, nil
}

// GeneratePresignedUploadURL generates a presigned URL for uploading a file
//
// CRITICAL: Presigned URL Security
// When ContentType is set in PutObjectInput, the AWS SDK automatically includes
// "content-type" in X-Amz-SignedHeaders. This means:
//   1. The frontend MUST send the exact same Content-Type header when uploading
//   2. The Content-Type value must match exactly what was used during signing
//   3. If the frontend sends ANY different headers or omits Content-Type, upload fails
//
// Example signed headers: X-Amz-SignedHeaders=content-type;host
//
// Why this matters:
//   - S3 signature v4 validates that signed headers match exactly
//   - Adding/removing/changing headers invalidates the signature
//   - This prevents tampering and ensures upload integrity
func (c *S3Client) GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)

	// Include ContentType in PutObjectInput to sign it explicitly
	// This ensures only clients with the correct Content-Type can upload
	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		// ContentType is included in signed headers (content-type;host)
		// Frontend MUST send: Content-Type: application/pdf (or whatever was specified)
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return request.URL, nil
}

// GeneratePresignedDownloadURL generates a presigned URL for downloading a file
func (c *S3Client) GeneratePresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return request.URL, nil
}

// DeleteObject deletes an object from S3
func (c *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// ObjectExists checks if an object exists in S3
func (c *S3Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if error is "NotFound"
		return false, nil
	}

	return true, nil
}
