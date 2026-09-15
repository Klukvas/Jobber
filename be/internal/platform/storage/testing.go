package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestS3Options reproduces the awkward answers real S3-compatible stores give,
// so the size-resolution paths can be exercised without one.
//
// Every field is keyed by object key and empty by default, which leaves the
// double behaving exactly as it did before any of them existed.
type TestS3Options struct {
	// ContentRangeOverride replaces the Content-Range value on a ranged (206)
	// response. An empty string omits the header entirely.
	ContentRangeOverride map[string]string
	// OmitContentLength answers a whole-object request with a chunked body, so
	// the response carries no Content-Length at all.
	OmitContentLength map[string]bool
	// IgnoreRange serves the whole object even when a Range was asked for —
	// some stores do exactly this.
	IgnoreRange map[string]bool
	// FailHead makes HeadObject fail with a server error for this key.
	FailHead map[string]bool
}

// NewTestS3Client creates an S3Client backed by a local HTTP test server.
// The returned cleanup function must be called to shut down the server.
// The getObjectData map controls what data is returned for each key; DELETE
// removes the entry from that map, so tests can assert a rejected upload was
// actually cleaned up.
func NewTestS3Client(getObjectData map[string][]byte) (*S3Client, func()) {
	return NewTestS3ClientWithOptions(getObjectData, TestS3Options{})
}

// NewTestS3ClientWithOptions is NewTestS3Client with the response quirks in
// TestS3Options turned on.
func NewTestS3ClientWithOptions(getObjectData map[string][]byte, opts TestS3Options) (*S3Client, func()) {
	// net/http serves every request on its own goroutine, so the object map is
	// touched concurrently the moment a test drives more than one request at a
	// time — which the plan-limit tests do on purpose. Guarded rather than
	// documented away: an unsynchronised map here fails the race detector for
	// reasons that have nothing to do with the code under test.
	var mu sync.Mutex
	readObject := func(key string) ([]byte, bool) {
		mu.Lock()
		defer mu.Unlock()
		data, ok := getObjectData[key]
		return data, ok
	}
	deleteObject := func(key string) {
		mu.Lock()
		defer mu.Unlock()
		delete(getObjectData, key)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// S3 GetObject uses GET /{bucket}/{key}
		// With path-style: GET /test-bucket/some/key
		key := r.URL.Path
		// Strip leading /test-bucket/
		if len(key) > len("/test-bucket/") {
			key = key[len("/test-bucket/"):]
		}

		if r.Method == http.MethodDelete {
			deleteObject(key)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		data, ok := readObject(key)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code></Error>`)
			return
		}

		// HeadObject: the size, no body. Answered before the range handling
		// below because a HEAD never carries one.
		if r.Method == http.MethodHead {
			if opts.FailHead[key] {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")

		// Honour a "bytes=0-N" range so GetObjectPrefix can be exercised the
		// way real storage answers it.
		rangeHeader := r.Header.Get("Range")
		if strings.HasPrefix(rangeHeader, "bytes=0-") && !opts.IgnoreRange[key] {
			if end, err := strconv.Atoi(strings.TrimPrefix(rangeHeader, "bytes=0-")); err == nil {
				// A present but zero-byte object cannot satisfy any range, and
				// real S3/Hetzner answer 416 InvalidRange. Emulating that
				// matters: this double used to reply 206 with a malformed
				// "bytes 0--1/0" header, which hid the fact that production
				// took the error path here.
				if len(data) == 0 {
					w.Header().Set("Content-Range", "bytes */0")
					w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>InvalidRange</Code></Error>`)
					return
				}
				last := min(end, len(data)-1)
				chunk := data[:last+1]
				contentRange := fmt.Sprintf("bytes 0-%d/%d", last, len(data))
				if override, overridden := opts.ContentRangeOverride[key]; overridden {
					contentRange = override
				}
				if contentRange != "" {
					w.Header().Set("Content-Range", contentRange)
				}
				w.WriteHeader(http.StatusPartialContent)
				w.Write(chunk)
				return
			}
		}

		if opts.OmitContentLength[key] {
			// Flushing the headers first forces a chunked body, which is the
			// only way to answer without a Content-Length.
			w.WriteHeader(http.StatusOK)
			if flusher, canFlush := w.(http.Flusher); canFlush {
				flusher.Flush()
			}
			w.Write(data)
			return
		}

		// Set explicitly: net/http only infers Content-Length for bodies small
		// enough to buffer, and a multi-megabyte object would otherwise go out
		// chunked and silently exercise the no-length path.
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
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
		// One attempt: the double's failure cases are deliberate, and the
		// default three-with-backoff turns each of them into seconds of test.
		o.RetryMaxAttempts = 1
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
