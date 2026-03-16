package pdf

import (
	"bytes"
	"fmt"
	"strings"

	pdflib "github.com/ledongthuc/pdf"
)

// MaxPDFSize is the maximum allowed PDF file size (5MB).
const MaxPDFSize = 5 * 1024 * 1024

// ExtractText extracts plain text from PDF bytes.
func ExtractText(pdfBytes []byte) (string, error) {
	if len(pdfBytes) > MaxPDFSize {
		return "", fmt.Errorf("PDF file exceeds maximum size of %d bytes", MaxPDFSize)
	}

	reader := bytes.NewReader(pdfBytes)
	pdfReader, err := pdflib.NewReader(reader, int64(len(pdfBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	var sb strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip unreadable pages
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("no text could be extracted from the PDF")
	}

	return result, nil
}
