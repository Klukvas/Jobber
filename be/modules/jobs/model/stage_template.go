package model

import "time"

// StageTemplate represents a reusable stage definition
type StageTemplate struct {
	ID        string
	UserID    string
	Name      string
	Order     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StageTemplateDTO represents stage template data transfer object
type StageTemplateDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Order     int       `json:"order"`
	CreatedAt time.Time `json:"created_at"`
}

// ToDTO converts StageTemplate to StageTemplateDTO
func (s *StageTemplate) ToDTO() *StageTemplateDTO {
	return &StageTemplateDTO{
		ID:        s.ID,
		Name:      s.Name,
		Order:     s.Order,
		CreatedAt: s.CreatedAt,
	}
}
