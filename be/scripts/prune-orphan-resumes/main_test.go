package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The whole tool turns on this one decision, and both halves of it protect
// something: the reference check protects a customer's resume, the age check
// protects an upload whose finalize call has not landed yet.
func TestIsOrphan(t *testing.T) {
	cutoff := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	live := "users/u1/resumes/live.pdf"
	referenced := map[string]struct{}{live: {}}

	tests := []struct {
		name    string
		obj     candidate
		want    bool
		wantWhy string
	}{
		{
			name:    "an abandoned upload past the window",
			obj:     candidate{key: "users/u1/resumes/abandoned.pdf", modified: cutoff.Add(-time.Hour)},
			want:    true,
			wantWhy: "nothing points at it and nothing is going to",
		},
		{
			name:    "a resume a row points at",
			obj:     candidate{key: live, modified: cutoff.Add(-365 * 24 * time.Hour)},
			want:    false,
			wantWhy: "age is irrelevant for an object that is in use",
		},
		{
			name:    "an upload being finalized right now",
			obj:     candidate{key: "users/u1/resumes/in-flight.pdf", modified: cutoff.Add(time.Minute)},
			want:    false,
			wantWhy: "it has no row yet either, and deleting it would break a live request",
		},
		{
			name:    "an object exactly on the cutoff",
			obj:     candidate{key: "users/u1/resumes/edge.pdf", modified: cutoff},
			want:    false,
			wantWhy: "the window is inclusive on the safe side",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isOrphan(tt.obj, referenced, cutoff), tt.wantWhy)
		})
	}
}

func TestHumanBytes(t *testing.T) {
	assert.Equal(t, "512 B", humanBytes(512))
	assert.Equal(t, "1.0 KB", humanBytes(1024))
	assert.Equal(t, "2.5 MB", humanBytes(2621440))
}
