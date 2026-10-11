package model

import "time"

// StageTemplate is a user-defined pipeline column. The user's ordered set of
// templates IS the pipeline; a job sits in exactly one of them. There is no
// phase and no status — the column is the state. Kind carries the column's
// semantic meaning for analytics, so name and order stay purely the user's.
type StageTemplate struct {
	ID        string
	UserID    string
	Name      string
	Order     int
	Kind      StageKind
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StageTemplateDTO represents stage template data transfer object
type StageTemplateDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Order     int       `json:"order"`
	Kind      StageKind `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

// EffectiveKind is the column's semantic kind, with anything unrecognized —
// an unset field on a row written before the kind column existed, or a value
// from a newer schema — normalized to the unclassified default. Every decision
// that reads a kind goes through here, so a legacy row behaves like a custom
// column instead of matching no case at all.
func (s *StageTemplate) EffectiveKind() StageKind {
	if !s.Kind.IsValid() {
		return StageKindOther
	}
	return s.Kind
}

// ToDTO converts StageTemplate to StageTemplateDTO. An unset kind is reported
// as "other" so the field is always a valid enum value on the wire, even for a
// row written before the kind column existed.
func (s *StageTemplate) ToDTO() *StageTemplateDTO {
	return &StageTemplateDTO{
		ID:        s.ID,
		Name:      s.Name,
		Order:     s.Order,
		Kind:      s.EffectiveKind(),
		CreatedAt: s.CreatedAt,
	}
}
