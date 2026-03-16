package model

import "time"

// ContentLibraryEntry represents a saved content snippet.
type ContentLibraryEntry struct {
	ID        string
	UserID    string
	Title     string
	Content   string
	Category  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ContentLibraryEntryDTO is the JSON response for a content library entry.
type ContentLibraryEntryDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToDTO converts a ContentLibraryEntry to ContentLibraryEntryDTO.
func (e *ContentLibraryEntry) ToDTO() *ContentLibraryEntryDTO {
	return &ContentLibraryEntryDTO{
		ID:        e.ID,
		Title:     e.Title,
		Content:   e.Content,
		Category:  e.Category,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// CreateContentLibraryRequest is the request for creating a content library entry.
type CreateContentLibraryRequest struct {
	Title    string `json:"title" binding:"required,max=255"`
	Content  string `json:"content" binding:"required"`
	Category string `json:"category" binding:"max=50"`
}

// UpdateContentLibraryRequest is the request for updating a content library entry.
type UpdateContentLibraryRequest struct {
	Title    *string `json:"title" binding:"omitempty,max=255"`
	Content  *string `json:"content"`
	Category *string `json:"category" binding:"omitempty,max=50"`
}
