package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasCoverLetterCandidateSource(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{name: "empty"},
		{name: "empty object", source: `{}`},
		{name: "null", source: `null`},
		{name: "raw text", source: `Backend engineer with Go`},
		{name: "blank work", source: `{"experiences":[{"position":"Engineer","description":" "}]}`},
		{name: "work", source: `{"experiences":[{"position":"Engineer","description":"Built Go APIs."}]}`, want: true},
		{name: "summary fallback", source: `{"summary":{"content":"Computer science student."}}`, want: true},
		{name: "credential fallback", source: `{"educations":[{"institution":"University","degree":"Studying CS"}]}`, want: true},
		{name: "skill fallback", source: `{"skills":[{"name":"Go"}]}`, want: true},
		{name: "oversized", source: `{"summary":{"content":"` + strings.Repeat("x", 4001) + `"}}`},
		{name: "NUL", source: `{"summary":{"content":"Built\u0000 APIs"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, HasCoverLetterCandidateSource(tc.source)) })
	}
}
