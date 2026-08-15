package model

import "time"

// StageTemplate is a user-defined pipeline column. The user's ordered set of
// templates IS the pipeline; a job sits in exactly one of them. There is no
// phase and no status — the column is the state.
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
