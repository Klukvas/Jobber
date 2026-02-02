package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("S3_BUCKET")
	region := os.Getenv("S3_REGION")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")

	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		log.Fatal("Missing S3 configuration in .env file")
	}

	fmt.Printf("Setting up CORS for bucket: %s\n", bucket)

	// Create custom endpoint resolver
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// Create AWS config
	cfg := aws.Config{
		Region:                      region,
		Credentials:                 credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		EndpointResolverWithOptions: customResolver,
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Define CORS rules
	corsConfig := &types.CORSConfiguration{
		CORSRules: []types.CORSRule{
			{
				AllowedOrigins: []string{
					"http://localhost:5173",  // Vite dev server
					"http://localhost:3000",  // Common React dev server
					"http://localhost:8080",  // Backend (for testing)
					"https://your-production-domain.com", // Add your production domain
				},
				AllowedMethods: []string{
					"GET",
					"PUT",
					"POST",
					"DELETE",
					"HEAD",
				},
				AllowedHeaders: []string{
					"*", // Allow all headers
				},
				MaxAgeSeconds: aws.Int32(3000),
			},
		},
	}

	// Apply CORS configuration
	ctx := context.Background()
	_, err := client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket:            aws.String(bucket),
		CORSConfiguration: corsConfig,
	})

	if err != nil {
		log.Fatalf("Failed to set CORS configuration: %v", err)
	}

	fmt.Println("✅ CORS configuration applied successfully!")
	fmt.Println("\nAllowed origins:")
	for _, origin := range corsConfig.CORSRules[0].AllowedOrigins {
		fmt.Printf("  - %s\n", origin)
	}
	fmt.Println("\nAllowed methods:")
	for _, method := range corsConfig.CORSRules[0].AllowedMethods {
		fmt.Printf("  - %s\n", method)
	}
}
