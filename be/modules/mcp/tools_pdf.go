package mcp

import (
	"bytes"
	"context"
	"time"
)

const pdfLinkLifetime = 15 * time.Minute

type ResumePDFInput struct {
	ResumeID string `json:"resume_id" jsonschema:"UUID of your Builder Resume to export"`
}
type ResumePDFOutput struct {
	URL       string `json:"url" jsonschema:"Private presigned PDF download URL, valid for 15 minutes"`
	ExpiresAt string `json:"expires_at" jsonschema:"URL expiration timestamp in UTC RFC3339 format"`
}

func (t *tools) getResumePDFLink(ctx context.Context, owner string, input ResumePDFInput) (ResumePDFOutput, error) {
	id, err := canonicalID(input.ResumeID, "resume_id")
	if err != nil {
		return ResumePDFOutput{}, err
	}
	if t.ai.Builders == nil || t.ai.PDF == nil || t.ai.Storage == nil || !t.ai.PDF.HasFrontendPDF() {
		return ResumePDFOutput{}, errPDFUnavailable
	}
	resume, err := t.ai.Builders.Get(ctx, owner, id)
	if err != nil {
		return ResumePDFOutput{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	data, err := t.ai.PDF.GenerateResumePDFFromFrontend(ctx, resume)
	if err != nil {
		return ResumePDFOutput{}, errPDFUnavailable
	}
	key := "mcp-exports/" + owner + "/" + id + ".pdf"
	if err := t.ai.Storage.UploadPDF(ctx, key, bytes.NewReader(data)); err != nil {
		return ResumePDFOutput{}, errPDFUnavailable
	}
	// Start expiry before signing so the stated expiry never exceeds S3's lifetime.
	expires := time.Now().UTC().Add(pdfLinkLifetime)
	link, err := t.ai.Storage.GeneratePresignedDownloadURL(ctx, key, "resume.pdf", pdfLinkLifetime)
	if err != nil {
		return ResumePDFOutput{}, errPDFUnavailable
	}
	return ResumePDFOutput{URL: link, ExpiresAt: expires.Format(time.RFC3339)}, nil
}
