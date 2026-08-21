package ogimage

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decode(t *testing.T, data []byte) {
	t.Helper()
	require.NotEmpty(t, data)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, width, img.Bounds().Dx())
	assert.Equal(t, height, img.Bounds().Dy())
}

func TestRender_PopulatedSnapshot(t *testing.T) {
	snap := model.StatsSnapshot{
		Overview: model.OverviewSnapshot{
			TotalApplications:  77,
			ActiveApplications: 40,
			ResponseRate:       14,
		},
		Funnel: []model.FunnelStageSnapshot{
			{StageName: "Applied", StageOrder: 1, Count: 77},
			{StageName: "Screening", StageOrder: 2, Count: 22},
			{StageName: "Interview with a very long stage name", StageOrder: 3, Count: 9},
			{StageName: "Offer", StageOrder: 4, Count: 3},
			{StageName: "Beyond the max rows", StageOrder: 5, Count: 1},
		},
	}
	out, err := Render(snap)
	require.NoError(t, err)
	decode(t, out)
}

func TestRender_EmptySnapshot(t *testing.T) {
	out, err := Render(model.StatsSnapshot{})
	require.NoError(t, err)
	decode(t, out)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short", 20))
	assert.Equal(t, "abcd…", truncate("abcdefghij", 5))
	assert.Equal(t, "a", truncate("abc", 1))
}
