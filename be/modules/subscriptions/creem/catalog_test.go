package creem

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModeMatches(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		live, want bool
	}{
		{name: "test", mode: "test", want: true}, {name: "sandbox", mode: "sandbox", want: true}, {name: "live", mode: "prod", live: true, want: true},
		{name: "test cannot grant live", mode: "test", live: true}, {name: "sandbox cannot grant live", mode: "sandbox", live: true}, {name: "live cannot grant test", mode: "prod"},
		{name: "missing"}, {name: "unknown", mode: "local"},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, ModeMatches(tc.mode, tc.live)) })
	}
}
