package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewTestS3Client creates an S3Client backed by a local HTTP test server.
// The returned cleanup function must be called to shut down the server.
// The getObjectData map controls what data is returned for each key.
func NewTestS3Client(getObjectData map[string][]byte) (*S3Client, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// S3 GetObject uses GET /{bucket}/{key}
		// With path-style: GET /test-bucket/some/key
		key := r.URL.Path
		// Strip leading /test-bucket/
		if len(key) > len("/test-bucket/") {
			key = key[len("/test-bucket/"):]
		}

		data, ok := getObjectData[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code></Error>`)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               srv.URL,
			SigningRegion:     "us-east-1",
			HostnameImmutable: true,
		}, nil
	})

	awsCfg := aws.Config{
		Region:                      "us-east-1",
		Credentials:                 credentials.NewStaticCredentialsProvider("test", "test", ""),
		EndpointResolverWithOptions: customResolver,
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Client{
			client: s3Client,
			bucket: "test-bucket",
		}, func() {
			srv.Close()
		}
}

// GetObjectForTest wraps GetObject for testing convenience.
func (c *S3Client) GetObjectForTest(ctx context.Context, key string) ([]byte, error) {
	return c.GetObject(ctx, key)
}
