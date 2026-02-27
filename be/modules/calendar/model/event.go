package model

import "time"

// CreateEventRequest represents a request to create a calendar event
type CreateEventRequest struct {
	StageID     string `json:"stage_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	StartTime   string `json:"start_time" binding:"required"`
	DurationMin int    `json:"duration_min" binding:"required,min=15,max=480"`
	Description string `json:"description,omitempty"`
}

// CalendarEventDTO represents a calendar event response
type CalendarEventDTO struct {
	EventID   string    `json:"event_id"`
	StageID   string    `json:"stage_id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Link      string    `json:"link,omitempty"`
}

// OAuthURLResponse represents the OAuth redirect URL
type OAuthURLResponse struct {
	URL string `json:"url"`
}
