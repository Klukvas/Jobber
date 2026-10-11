package service

import (
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
)

// When a card counts as "applied to" — the one rule Create, Move, AddStage and
// UpdateStage all decide applied_at and the initial history row from.

// columnImpliesApplied reports whether a card landing in this column has
// applied to the role.
//
// The kind decides it, not the position. "Anything past the first column means
// applied" was wrong in both directions as soon as the user touched their
// board: dragging Applied to the front (order 0) stopped stamping the date,
// and dragging Wishlist to the middle started stamping it for a card that was
// only bookmarked.
//
// An unclassified column ('other', and legacy rows written before kinds
// existed) has no meaning to read, so it keeps the old positional rule. That is
// deliberate compatibility, not a second source of truth: a custom column the
// user has not classified behaves exactly as it did before kinds, and
// classifying it in the stage UI takes over from the position. Every seeded
// column already carries a kind (migration 000047 backfills them), so this
// branch only covers columns the user invented.
func columnImpliesApplied(column *model.StageTemplate) bool {
	kind := column.EffectiveKind()
	switch {
	case kind.ImpliesApplied():
		return true
	case kind == model.StageKindSaved:
		return false
	default:
		return column.Order > 0
	}
}

// appliedAtAfterMove returns the applied_at to store when a card lands in
// target.
//
// An existing timestamp is always kept: it is a date the user may have picked
// themselves, and a move is never a reason to re-stamp or clear it. Moving a
// card back into the wishlist therefore does not erase the fact that it was
// applied to — only the card's position changes.
//
// current is the value read under the row lock, so a concurrent move cannot
// make this stamp a second time.
func appliedAtAfterMove(current *time.Time, target *model.StageTemplate, now time.Time) *time.Time {
	if current != nil {
		return current
	}
	if columnImpliesApplied(target) {
		return &now
	}
	return nil
}
