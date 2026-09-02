package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/netsafe"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MaxResumeDownloadSize caps resume file downloads from S3 or external URLs.
const MaxResumeDownloadSize = 20 * 1024 * 1024 // 20 MB

// ErrResumeFileMissing is returned when a resume record has neither an S3
// object nor an external file URL to download.
var ErrResumeFileMissing = errors.New("resume has no downloadable file")

// DownloadResumeFile retrieves resume file bytes from S3 (storageType "s3")
// or an external URL, with SSRF protection and a size cap. Takes primitive
// params so platform code stays independent of module models — callers pass
// string(resume.StorageType), resume.StorageKey, resume.FileURL.
func DownloadResumeFile(ctx context.Context, s3Client *S3Client, storageType string, storageKey, fileURL *string) ([]byte, error) {
	if storageType == "s3" && storageKey != nil && s3Client != nil {
		data, err := s3Client.GetObject(ctx, *storageKey)
		if err != nil {
			// A missing object usually means an abandoned presigned-upload
			// slot: the resume row is created before the browser PUTs the
			// file. Surface it as "no file" (callers map it to a friendly
			// 4xx) rather than an internal failure.
			var noKey *types.NoSuchKey
			if errors.As(err, &noKey) {
				return nil, ErrResumeFileMissing
			}
			return nil, fmt.Errorf("failed to download resume from S3: %w", err)
		}
		return data, nil
	}

	if fileURL != nil && *fileURL != "" {
		data, err := DownloadFileFromURL(ctx, *fileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download resume from URL: %w", err)
		}
		return data, nil
	}

	return nil, ErrResumeFileMissing
}

// DownloadFileFromURL fetches a file from an external URL with SSRF
// protection, a 30s timeout, and a MaxResumeDownloadSize cap.
func DownloadFileFromURL(ctx context.Context, rawURL string) ([]byte, error) {
	if err := netsafe.ValidateExternalURL(rawURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := netsafe.SafeClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, MaxResumeDownloadSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	if len(data) > MaxResumeDownloadSize {
		return nil, fmt.Errorf("resume file too large (max %d bytes)", MaxResumeDownloadSize)
	}

	return data, nil
}
