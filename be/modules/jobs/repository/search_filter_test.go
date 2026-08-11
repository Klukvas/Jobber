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

	t.Run("placeholder index follows already-appended args (e.g. after status filter)", func(t *testing.T) {
		// Simulate statusFilter having appended a status arg first.
		args := []any{"user-123", "applied"}
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

func TestStatusAndSearchFilterCompose(t *testing.T) {
	// statusFilter appends the status arg at $2, searchFilter then uses $3.
	args := []any{"user-123"}
	statusFrag := statusFilter("applied", &args)
	searchFrag := searchFilter("vercel", &args)

	assert.Equal(t, " AND j.status = $2", statusFrag)
	assert.Equal(t, " AND (j.title ILIKE $3 OR c.name ILIKE $3)", searchFrag)
	assert.Equal(t, []any{"user-123", "applied", "%vercel%"}, args)
}
