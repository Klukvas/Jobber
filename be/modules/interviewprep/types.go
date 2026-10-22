// Package interviewprep owns durable, Job-specific interview preparation.
package interviewprep

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrCapacity      = errors.New("interview answer capacity exceeded")
	ErrInputTooLarge = errors.New("interview source capacity exceeded")
	ErrFenced        = errors.New("interview lease lost")
	ErrValidation    = errors.New("invalid interview preparation input")
	ErrNotFound      = errors.New("interview preparation resource not found")
	ErrConflict      = errors.New("interview preparation conflict")
	ErrSourceChanged = errors.New("interview source changed")
	ErrUnavailable   = errors.New("interview AI unavailable")
	ErrRateLimited   = errors.New("interview operation rate exceeded")
	ErrInvalidAI     = errors.New("invalid interview AI response")
)

type Fact struct {
	ID      string `json:"id"`
	Section string `json:"section"`
	Text    string `json:"text"`
}
type Evidence struct {
	FactID string `json:"fact_id"`
	Quote  string `json:"quote"`
}
type STAR struct {
	Situation string `json:"situation"`
	Task      string `json:"task"`
	Action    string `json:"action"`
	Result    string `json:"result"`
}
type Question struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Question   string     `json:"question"`
	Purpose    string     `json:"purpose"`
	Difficulty string     `json:"difficulty"`
	Advice     string     `json:"advice"`
	Evidence   []Evidence `json:"evidence"`
	STAR       *STAR      `json:"star"`
}
type Requirement struct {
	Requirement string     `json:"requirement"`
	Assessment  string     `json:"assessment"`
	Evidence    []Evidence `json:"evidence"`
}
type Gap struct {
	Priority string `json:"priority"`
	Gap      string `json:"gap"`
	Action   string `json:"action"`
}
type Plan struct {
	Requirements         []Requirement `json:"requirements"`
	Questions            []Question    `json:"questions"`
	Gaps                 []Gap         `json:"gaps"`
	InterviewerQuestions []string      `json:"interviewer_questions"`
}
type Criterion struct {
	Name        string     `json:"name"`
	Score       *int       `json:"score"`
	Explanation string     `json:"explanation"`
	Evidence    []Evidence `json:"evidence"`
}
type Followup struct {
	Question string `json:"question"`
	Purpose  string `json:"purpose"`
}
type Review struct {
	Criteria     []Criterion `json:"criteria"`
	Strengths    []string    `json:"strengths"`
	Improvements []string    `json:"improvements"`
	Revisit      []string    `json:"revisit"`
	Followup     *Followup   `json:"followup"`
}
type Version struct {
	ID              string    `json:"id"`
	JobID           string    `json:"job_id"`
	ResumeBuilderID string    `json:"resume_builder_id"`
	ResumeTitle     string    `json:"resume_title"`
	JobTitle        string    `json:"job_title"`
	Locale          string    `json:"locale"`
	CreatedAt       time.Time `json:"created_at"`
	Plan            Plan      `json:"plan"`
	Facts           []Fact    `json:"facts"`
	Studied         []string  `json:"studied"`
	Stale           bool      `json:"stale"`
	SourceHash      string    `json:"-"`
}
type Answer struct {
	ID         string    `json:"id"`
	QuestionID string    `json:"question_id"`
	Text       string    `json:"text"`
	Skipped    bool      `json:"skipped"`
	Feedback   *Review   `json:"feedback"`
	CreatedAt  time.Time `json:"created_at"`
}
type Session struct {
	AnswerBytesRemaining int        `json:"answer_bytes_remaining"`
	ID                   string     `json:"id"`
	VersionID            string     `json:"version_id"`
	JobID                string     `json:"job_id"`
	Status               string     `json:"status"`
	Revision             int        `json:"revision"`
	Questions            []Question `json:"questions"`
	Answers              []Answer   `json:"answers"`
	Summary              *Review    `json:"summary"`
	CreatedAt            time.Time  `json:"created_at"`
}
type Operation struct {
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	Status          string          `json:"status"`
	ErrorCode       string          `json:"error_code"`
	Attempts        int             `json:"attempts"`
	CreatedAt       time.Time       `json:"created_at"`
	VersionID       *string         `json:"version_id"`
	SessionID       *string         `json:"session_id"`
	AnswerID        *string         `json:"answer_id"`
	Owner           string          `json:"-"`
	JobID           string          `json:"-"`
	Fingerprint     string          `json:"-"`
	ProviderStarted bool            `json:"-"`
	ChargeReleased  bool            `json:"-"`
	Token           string          `json:"-"`
	Input           json.RawMessage `json:"-"`
}
type Snapshot struct {
	JobID       string `json:"job_id"`
	ResumeID    string `json:"resume_id"`
	JobTitle    string `json:"job_title"`
	ResumeTitle string `json:"resume_title"`
	Job         string `json:"job"`
	Facts       []Fact `json:"facts"`
	Locale      string `json:"locale"`
	Hash        string `json:"hash"`
}
type AIAnswer struct {
	ID         string `json:"id"`
	QuestionID string `json:"question_id"`
	Text       string `json:"text"`
	Skipped    bool   `json:"skipped"`
}
type QuestionContext struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Question string `json:"question"`
	Purpose  string `json:"purpose"`
}
type AIInput struct {
	Snapshot  Snapshot          `json:"snapshot"`
	Question  *QuestionContext  `json:"question,omitempty"`
	Questions []QuestionContext `json:"questions,omitempty"`
	Answers   []AIAnswer        `json:"answers,omitempty"`
}
type VersionSummary struct {
	ID              string    `json:"id"`
	JobID           string    `json:"job_id"`
	ResumeBuilderID string    `json:"resume_builder_id"`
	ResumeTitle     string    `json:"resume_title"`
	JobTitle        string    `json:"job_title"`
	Locale          string    `json:"locale"`
	CreatedAt       time.Time `json:"created_at"`
}
type SessionSummary struct {
	ID        string    `json:"id"`
	VersionID string    `json:"version_id"`
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"created_at"`
}
type History struct {
	Versions    []VersionSummary `json:"versions"`
	Sessions    []SessionSummary `json:"sessions"`
	Operations  []Operation      `json:"operations"`
	AIAvailable bool             `json:"ai_available"`
}
