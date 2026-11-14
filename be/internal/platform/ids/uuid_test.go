package ids

import (
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
)

const canonical = "0b8f7a7e-5d5c-4c38-9c3e-2f1e6a0d4b11"

func TestCanonicalUUID(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   string
		wantOK bool
	}{
		{name: "canonical stays unchanged", raw: canonical, want: canonical, wantOK: true},
		{name: "urn form", raw: "urn:uuid:" + canonical, want: canonical, wantOK: true},
		{name: "braces", raw: "{" + canonical + "}", want: canonical, wantOK: true},
		{name: "uppercase", raw: "0B8F7A7E-5D5C-4C38-9C3E-2F1E6A0D4B11", want: canonical, wantOK: true},
		{name: "compact", raw: "0b8f7a7e5d5c4c389c3e2f1e6a0d4b11", want: canonical, wantOK: true},
		{name: "empty", raw: "", wantOK: false},
		{name: "not a uuid", raw: "not-a-uuid", wantOK: false},
		{name: "path traversal", raw: "../" + canonical, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CanonicalUUID(tt.raw)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("CanonicalUUID(%q) = (%q, %v), want (%q, %v)", tt.raw, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// Other packages' regression tests rely on every fixture form canonicalising.
func TestNonCanonicalFormsFixture(t *testing.T) {
	forms := idstest.NonCanonicalForms(canonical)
	if len(forms) == 0 {
		t.Fatal("expected at least one non-canonical form")
	}
	for name, form := range forms {
		got, ok := CanonicalUUID(form)
		if !ok || got != canonical {
			t.Fatalf("%s form %q did not canonicalise: (%q, %v)", name, form, got, ok)
		}
		if form == canonical {
			t.Fatalf("%s form equals the canonical form", name)
		}
	}
}

func TestCanonicalUUIDs(t *testing.T) {
	const other = "0b8f7a7e-5d5c-4c38-9c3e-2f1e6a0d4b22"
	got, ok := CanonicalUUIDs(idstest.NonCanonicalForms(canonical)["urn"], idstest.NonCanonicalForms(other)["braces"])
	if !ok || len(got) != 2 || got[0] != canonical || got[1] != other {
		t.Fatalf("CanonicalUUIDs = (%q, %v), want ([%q %q], true)", got, ok, canonical, other)
	}
	if got, ok := CanonicalUUIDs(canonical, "nope"); ok || got != nil {
		t.Fatalf("CanonicalUUIDs with an invalid id = (%q, %v), want (nil, false)", got, ok)
	}
}
