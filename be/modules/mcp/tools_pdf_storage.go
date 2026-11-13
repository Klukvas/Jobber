package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/storage"
)

// S3PDFStorage reuses the existing presigned upload API without exposing uploads
// to the assistant. The same key is overwritten for every export of a resume.
type S3PDFStorage struct{ Client *storage.S3Client }

func (s S3PDFStorage) UploadPDF(ctx context.Context, key string, body io.Reader) error {
	link, err := s.Client.GeneratePresignedUploadURL(ctx, key, "application/pdf", pdfLinkLifetime)
	if err != nil {
		return fmt.Errorf("sign PDF upload: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, link, body)
	if err != nil {
		return fmt.Errorf("prepare PDF upload: %w", err)
	}
	request.Header.Set("Content-Type", "application/pdf")
	client := &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("upload PDF: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("PDF upload status %d", response.StatusCode)
	}
	return nil
}
func (s S3PDFStorage) GeneratePresignedDownloadURL(ctx context.Context, key, name string, expiry time.Duration) (string, error) {
	return s.Client.GeneratePresignedDownloadURL(ctx, key, name, expiry)
}
