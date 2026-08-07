package model

import "time"

// Phase is a fixed, system-defined pipeline zone that contains user stages.
// The five phases mirror the whole application lifecycle.
type Phase string

const (
	PhaseWishlist   Phase = "wishlist"
	PhaseApplied    Phase = "applied"
	PhaseInProgress Phase = "in_progress"
	PhaseOffer      Phase = "offer"
	PhaseRejected   Phase = "rejected"
)

// DefaultPhase is applied when a template is created without an explicit
// phase — the vast majority of user stages are interview steps.
const DefaultPhase = PhaseInProgress

// PhaseRank orders phases along the pipeline; used for composite sorting
// (phase first, then the template's own order).
var PhaseRank = map[Phase]int{
	PhaseWishlist:   0,
	PhaseApplied:    1,
	PhaseInProgress: 2,
	PhaseOffer:      3,
	PhaseRejected:   4,
}

func IsValidPhase(p Phase) bool {
	_, ok := PhaseRank[p]
	return ok
}

// StatusForPhase derives the job status implied by standing in a phase.
// The status enum itself is unchanged — it is simply a projection of the
// card's position on the unified board.
func StatusForPhase(p Phase) JobStatus {
	switch p {
	case PhaseWishlist:
		return StatusSaved
	case PhaseOffer:
		return StatusOffer
	case PhaseRejected:
		return StatusRejected
	default:
		return StatusApplied
	}
}

// StageTemplate represents a reusable stage definition
type StageTemplate struct {
	ID        string
	UserID    string
	Name      string
	Order     int
	Phase     Phase
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StageTemplateDTO represents stage template data transfer object
type StageTemplateDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Order     int       `json:"order"`
	Phase     Phase     `json:"phase"`
	CreatedAt time.Time `json:"created_at"`
}

// ToDTO converts StageTemplate to StageTemplateDTO
func (s *StageTemplate) ToDTO() *StageTemplateDTO {
	return &StageTemplateDTO{
		ID:        s.ID,
		Name:      s.Name,
		Order:     s.Order,
		Phase:     s.Phase,
		CreatedAt: s.CreatedAt,
	}
}
