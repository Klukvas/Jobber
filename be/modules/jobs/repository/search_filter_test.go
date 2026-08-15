package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchFilter(t *testing.T) {
	t.Run("empty search yields no fragment and no args", func(t *testing.T) {
		args := []any{"user-123"}
		frag := searchFilter("", &args)

		assert.Equal(t, "", frag)
		assert.Equal(t, []any{"user-123"}, args)
	})

	t.Run("whitespace-only search yields no fragment", func(t *testing.T) {
		args := []any{"user-123"}
		frag := searchFilter("   ", &args)

		assert.Equal(t, "", frag)
		assert.Len(t, args, 1)
	})

	t.Run("non-empty search appends wrapped term and uses next placeholder", func(t *testing.T) {
		args := []any{"user-123"}
		frag := searchFilter("Stripe", &args)

		assert.Equal(t, " AND (j.title ILIKE $2 OR c.name ILIKE $2)", frag)
		assert.Equal(t, []any{"user-123", "%Stripe%"}, args)
	})

	t.Run("placeholder index follows already-appended args", func(t *testing.T) {
		// Simulate a prior arg (e.g. a company id) having been appended first.
		args := []any{"user-123", "company-1"}
		frag := searchFilter("acme", &args)

		assert.Equal(t, " AND (j.title ILIKE $3 OR c.name ILIKE $3)", frag)
		assert.Equal(t, "%acme%", args[2])
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		args := []any{"user-123"}
		searchFilter("  google  ", &args)

		assert.Equal(t, "%google%", args[1])
	})

	t.Run("escapes LIKE wildcards so they match literally", func(t *testing.T) {
		args := []any{"user-123"}
		searchFilter(`50%_off\x`, &args)

		// % -> \%, _ -> \_, \ -> \\ ; then wrapped in %...%
		assert.Equal(t, `%50\%\_off\\x%`, args[1])
	})
}

func TestArchivedFilter(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		// "" and the legacy "active" mean "everything except archived"
		// (backward compatible with the deployed Chrome extension).
		{"empty excludes archived", "", " AND j.is_archived = false"},
		{"legacy active excludes archived", "active", " AND j.is_archived = false"},
		{"unknown value excludes archived", "applied", " AND j.is_archived = false"},
		{"archived includes only archived", "archived", " AND j.is_archived = true"},
		{"all imposes no filter", "all", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, archivedFilter(tt.status))
		})
	}
}

// TestArchivedAndSearchFilterCompose verifies the two WHERE fragments used by
// List cooperate: the archived filter takes no placeholder, so search still
// binds at $2.
func TestArchivedAndSearchFilterCompose(t *testing.T) {
	args := []any{"user-123"}
	archivedFrag := archivedFilter("")
	searchFrag := searchFilter("vercel", &args)

	assert.Equal(t, " AND j.is_archived = false", archivedFrag)
	assert.Equal(t, " AND (j.title ILIKE $2 OR c.name ILIKE $2)", searchFrag)
	assert.Equal(t, []any{"user-123", "%vercel%"}, args)
}
