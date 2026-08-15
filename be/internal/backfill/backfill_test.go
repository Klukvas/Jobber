package backfill

import "testing"

// columns is a representative name->id map for the canonical pipeline columns a
// user has after ensureColumns.
var columns = map[string]string{
	"Wishlist": "col-wishlist",
	"Applied":  "col-applied",
	"Offer":    "col-offer",
	"Rejected": "col-rejected",
}

func strptr(s string) *string { return &s }

func TestResolveTarget_ByStatus(t *testing.T) {
	cases := []struct {
		name            string
		status          string
		appliedAtIsNull bool
		want            string
	}{
		{"saved -> Wishlist", "saved", true, "col-wishlist"},
		{"applied -> Applied", "applied", false, "col-applied"},
		{"on_hold -> Applied", "on_hold", false, "col-applied"},
		{"offer -> Offer", "offer", false, "col-offer"},
		{"rejected -> Rejected", "rejected", false, "col-rejected"},
		{"archived + never applied -> Wishlist", "archived", true, "col-wishlist"},
		{"archived + was applied -> Applied", "archived", false, "col-applied"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			j := jobRow{id: "j1", oldStatus: c.status, appliedAtIsNull: c.appliedAtIsNull}
			got, err := resolveTarget(j, columns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("status %q: got %q, want %q", c.status, got, c.want)
			}
		})
	}
}

// A live current_stage pointer always wins, regardless of the legacy status —
// this is what preserves exactly where an in-progress card sat.
func TestResolveTarget_CurrentStageWins(t *testing.T) {
	j := jobRow{
		id:                     "j1",
		oldStatus:              "saved", // would otherwise map to Wishlist
		currentStageTemplateID: strptr("col-technical"),
	}
	got, err := resolveTarget(j, columns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "col-technical" {
		t.Errorf("current stage should win: got %q, want %q", got, "col-technical")
	}
}

// An empty-string pointer is treated as absent (falls through to status mapping).
func TestResolveTarget_EmptyPointerIgnored(t *testing.T) {
	j := jobRow{id: "j1", oldStatus: "offer", currentStageTemplateID: strptr("")}
	got, err := resolveTarget(j, columns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "col-offer" {
		t.Errorf("empty pointer should fall through to status: got %q, want %q", got, "col-offer")
	}
}

func TestResolveTarget_UnknownStatusErrors(t *testing.T) {
	j := jobRow{id: "j1", oldStatus: "bogus"}
	if _, err := resolveTarget(j, columns); err == nil {
		t.Error("expected error for unknown status, got nil")
	}
}

func TestStatsMerge(t *testing.T) {
	a := Stats{Users: 1, ColumnsCreated: 2, JobsUpdated: 3, StagesAppended: 4}
	a.merge(Stats{Users: 10, ColumnsCreated: 20, JobsUpdated: 30, StagesAppended: 40})
	want := Stats{Users: 11, ColumnsCreated: 22, JobsUpdated: 33, StagesAppended: 44}
	if a != want {
		t.Errorf("merge = %+v, want %+v", a, want)
	}
}
