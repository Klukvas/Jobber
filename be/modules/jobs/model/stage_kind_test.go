package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageKind_IsResponse(t *testing.T) {
	// "Response" is the employer coming back at all — a rejection counts, an
	// interview or offer implies one happened earlier. "Applied" explicitly
	// does not: the application being sent is not a reply.
	responded := map[StageKind]bool{
		StageKindSaved:     false,
		StageKindApplied:   false,
		StageKindResponse:  true,
		StageKindInterview: true,
		StageKindOffer:     true,
		StageKindRejected:  true,
		StageKindOther:     false,
	}

	for kind, want := range responded {
		t.Run(string(kind), func(t *testing.T) {
			assert.Equal(t, want, kind.IsResponse())
		})
	}
}

func TestStageKind_ImpliesApplied(t *testing.T) {
	// Everything from "application sent" onwards can only be reached by
	// applying — a rejection included. Saved means the opposite, and "other"
	// carries no meaning, so neither claims an application happened.
	applied := map[StageKind]bool{
		StageKindSaved:     false,
		StageKindApplied:   true,
		StageKindResponse:  true,
		StageKindInterview: true,
		StageKindOffer:     true,
		StageKindRejected:  true,
		StageKindOther:     false,
	}

	for kind, want := range applied {
		t.Run(string(kind), func(t *testing.T) {
			assert.Equal(t, want, kind.ImpliesApplied())
		})
	}

	assert.False(t, StageKind("").ImpliesApplied(), "an unset kind claims nothing")
}

func TestStageTemplate_EffectiveKind(t *testing.T) {
	assert.Equal(t, StageKindInterview,
		(&StageTemplate{Kind: StageKindInterview}).EffectiveKind())
	assert.Equal(t, StageKindOther,
		(&StageTemplate{}).EffectiveKind(), "a row written before kinds existed is unclassified")
	assert.Equal(t, StageKindOther,
		(&StageTemplate{Kind: StageKind("ghosted")}).EffectiveKind(), "an unknown value is never trusted")
}

func TestStageKind_IsValid(t *testing.T) {
	for _, kind := range AllStageKinds {
		assert.True(t, kind.IsValid(), "%s must be a known kind", kind)
	}
	assert.False(t, StageKind("").IsValid())
	assert.False(t, StageKind("in_progress").IsValid(), "the dropped phase values are not kinds")
	assert.False(t, StageKind("INTERVIEW").IsValid(), "kinds are lowercase on the wire")
}

func TestParseStageKind(t *testing.T) {
	t.Run("normalizes case and surrounding space", func(t *testing.T) {
		kind, err := ParseStageKind("  Interview ")
		require.NoError(t, err)
		assert.Equal(t, StageKindInterview, kind)
	})

	t.Run("treats an empty value as the unclassified default", func(t *testing.T) {
		kind, err := ParseStageKind("   ")
		require.NoError(t, err)
		assert.Equal(t, StageKindOther, kind)
	})

	t.Run("rejects an unknown kind", func(t *testing.T) {
		_, err := ParseStageKind("ghosted")
		assert.ErrorIs(t, err, ErrInvalidStageKind)
	})
}

func TestStageTemplate_ToDTO_Kind(t *testing.T) {
	t.Run("carries the stored kind", func(t *testing.T) {
		tmpl := &StageTemplate{ID: "s1", Name: "Loop", Order: 3, Kind: StageKindInterview}

		dto := tmpl.ToDTO()

		assert.Equal(t, StageKindInterview, dto.Kind)
	})

	t.Run("reports an unset kind as other so the field is always valid", func(t *testing.T) {
		tmpl := &StageTemplate{ID: "s1", Name: "Follow-up"}

		dto := tmpl.ToDTO()

		assert.Equal(t, StageKindOther, dto.Kind)

		// The wire format always carries the field — the frontend can rely on it.
		raw, err := json.Marshal(dto)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"kind":"other"`)
	})
}

func TestCreateStageTemplateRequest_KindValidation(t *testing.T) {
	valid := "interview"
	require.NoError(t, validateBindingTags(t, CreateStageTemplateRequest{Name: "Loop", Kind: &valid}))

	require.NoError(t, validateBindingTags(t, CreateStageTemplateRequest{Name: "Loop"}),
		"kind is optional on input")

	bogus := "ghosted"
	assert.Error(t, validateBindingTags(t, CreateStageTemplateRequest{Name: "Loop", Kind: &bogus}))
	assert.Error(t, validateBindingTags(t, UpdateStageTemplateRequest{Kind: &bogus}))
}
