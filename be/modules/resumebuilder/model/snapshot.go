package model

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrInvalidSnapshotSource      = errors.New("snapshot source must contain 1 to 20 characters")
	ErrInvalidSnapshotContent     = errors.New("this resume snapshot is incomplete or invalid and cannot be restored")
	ErrUnsupportedSnapshotVersion = errors.New("this resume snapshot uses an unsupported format and cannot be restored")
)

// ResumeSnapshot stores a private full copy of a Builder Resume.
type ResumeSnapshot struct {
	ID              string
	ResumeBuilderID string
	UserID          string
	Source          string
	Content         json.RawMessage
	CreatedAt       time.Time
}

// ResumeSnapshotDTO deliberately excludes resume content.
type ResumeSnapshotDTO struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}
