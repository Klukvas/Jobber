package model

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrRecognitionRequired  = errors.New("review and confirm the imported resume first")
	ErrPDFUnreadable        = errors.New("PDF has no readable text; paste resume text instead")
	ErrOperationConflict    = errors.New("request key reused with different input")
	ErrOperationPending     = errors.New("request is still running")
	ErrOperationInterrupted = errors.New("request interrupted; it will not be charged or executed again")
	ErrOperationDeleted     = errors.New("the result was deleted; this request cannot recreate it")
)

// WorkflowOperation survives deletion of documents, so retries cannot recreate
// deleted copies or turn document-slot reuse into free AI calls.
type WorkflowOperation struct {
	AIReceiptID string
	UserID      string
	Kind        string
	Key         string
	JobID       string
	Fingerprint string
	Status      string
	ObjectID    string
	Result      json.RawMessage
	ErrorCode   string
	CreatedAt   time.Time
}

type PDFImportRequest struct {
	RequestKey string `json:"request_key" binding:"required,uuid"`
	ResumeID   string `json:"resume_id" binding:"required,uuid"`
	Text       string `json:"text" binding:"max=100000"`
}

type PDFImportResult struct {
	ResumeBuilder *ResumeBuilderDTO `json:"resume_builder"`
	ImportID      string            `json:"import_id"`
	JobID         string            `json:"job_id"`
}
