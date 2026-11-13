package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/require"
)

func TestResumePDFDeterministicKeyAndExpiry(t *testing.T) {
	f := newFakeAITools()
	client := aiTestClient(t, f, userA)
	before := time.Now().UTC()
	for i := 0; i < 2; i++ {
		result := decodeOutput[ResumePDFOutput](t, callTool(t, client, "get_resume_pdf_link", ResumePDFInput{ResumeID: "urn:uuid:" + builderA}))
		require.Equal(t, "https://storage.example/private.pdf", result.URL)
		expires, err := time.Parse(time.RFC3339, result.ExpiresAt)
		require.NoError(t, err)
		require.WithinDuration(t, before.Add(15*time.Minute), expires, 2*time.Second)
	}
	require.Equal(t, []string{"mcp-exports/" + userA + "/" + builderA + ".pdf", "mcp-exports/" + userA + "/" + builderA + ".pdf"}, f.uploadKeys)
	require.Equal(t, 15*time.Minute, f.expiry)
	require.Equal(t, []byte("%PDF fake"), f.uploaded)
	require.Zero(t, f.providerCalls)
	require.Zero(t, f.ledger.finished)
}

type failingPDFRenderer struct{ PDFRenderer }

func (f failingPDFRenderer) HasFrontendPDF() bool { return true }
func (f failingPDFRenderer) GenerateResumePDFFromFrontend(context.Context, *rbModel.FullResumeDTO) ([]byte, error) {
	return nil, errors.New("private renderer detail")
}
func TestResumePDFRendererFailureDoesNotUploadOrReturnLink(t *testing.T) {
	f := newFakeAITools()
	deps := f.deps()
	deps.PDF = failingPDFRenderer{}
	jobs := newFakeJobs()
	client := toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(deps)), userA)
	result := callTool(t, client, "get_resume_pdf_link", ResumePDFInput{ResumeID: builderA})
	assertPDFExportFailure(t, result)
	require.Empty(t, f.uploadKeys)
}
func TestResumePDFPresignFailureDoesNotReturnLink(t *testing.T) {
	f := newFakeAITools()
	failing := &pdfSigningFailure{PDFStorage: f, fail: true}
	result := callTool(t, pdfStorageClient(t, f, failing), "get_resume_pdf_link", ResumePDFInput{ResumeID: builderA})
	assertPDFExportFailure(t, result)
	require.Len(t, f.uploadKeys, 1)
	require.Equal(t, 1, failing.signCalls)
}
