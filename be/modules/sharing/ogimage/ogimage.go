// Package ogimage renders the 1200x630 Open Graph preview image for a shared
// stats snapshot — the picture social platforms show in the link card.
package ogimage

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	width  = 1200
	height = 630
	pad    = 64.0

	colorBg     = "#0B1120"
	colorAccent = "#A3E635"
	colorTile   = "#111A2E"
	colorTrack  = "#1E293B"
	colorText   = "#FFFFFF"
	colorMuted  = "#94A3B8"
	colorFooter = "#64748B"

	maxFunnelRows = 4
)

// Parsed once — the embedded Go fonts never fail to parse, but guard anyway.
var (
	fontBold    *truetype.Font
	fontRegular *truetype.Font
)

func init() {
	fontBold, _ = truetype.Parse(gobold.TTF)
	fontRegular, _ = truetype.Parse(goregular.TTF)
}

func face(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{Size: size})
}

// Render draws the OG preview for a snapshot and returns PNG bytes.
func Render(s model.StatsSnapshot) ([]byte, error) {
	if fontBold == nil || fontRegular == nil {
		return nil, fmt.Errorf("ogimage: fonts unavailable")
	}

	dc := gg.NewContext(width, height)
	dc.SetHexColor(colorBg)
	dc.Clear()

	// Accent rail on the left edge.
	dc.SetHexColor(colorAccent)
	dc.DrawRectangle(0, 0, 12, height)
	dc.Fill()

	drawHeader(dc)
	drawTiles(dc, s.Overview)
	drawFunnel(dc, s.Funnel)

	dc.SetFontFace(face(fontRegular, 24))
	dc.SetHexColor(colorFooter)
	dc.DrawStringAnchored("Tracked with Jobber", pad, height-38, 0, 0.5)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("ogimage: encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func drawHeader(dc *gg.Context) {
	dc.SetFontFace(face(fontBold, 40))
	dc.SetHexColor(colorAccent)
	dc.DrawStringAnchored("Jobber", pad, 78, 0, 0.5)

	dc.SetFontFace(face(fontRegular, 24))
	dc.SetHexColor(colorMuted)
	dc.DrawStringAnchored("jobber-app.com", width-pad, 78, 1, 0.5)

	dc.SetFontFace(face(fontBold, 46))
	dc.SetHexColor(colorText)
	dc.DrawStringAnchored("My job search", pad, 150, 0, 0.5)
}

func drawTiles(dc *gg.Context, ov model.OverviewSnapshot) {
	tiles := []struct{ value, label string }{
		{fmt.Sprintf("%d", ov.TotalApplications), "Applications"},
		{fmt.Sprintf("%.0f%%", ov.ResponseRate), "Response rate"},
		{fmt.Sprintf("%d", ov.ActiveApplications), "In progress"},
	}

	const tileH, gap = 150.0, 24.0
	tileW := (width-2*pad-2*gap)/3.0
	y := 196.0

	for i, t := range tiles {
		x := pad + float64(i)*(tileW+gap)

		dc.SetHexColor(colorTile)
		dc.DrawRoundedRectangle(x, y, tileW, tileH, 20)
		dc.Fill()

		dc.SetFontFace(face(fontBold, 68))
		dc.SetHexColor(colorAccent)
		dc.DrawStringAnchored(t.value, x+28, y+64, 0, 0.5)

		dc.SetFontFace(face(fontRegular, 26))
		dc.SetHexColor(colorMuted)
		dc.DrawStringAnchored(t.label, x+28, y+114, 0, 0.5)
	}
}

func drawFunnel(dc *gg.Context, funnel []model.FunnelStageSnapshot) {
	if len(funnel) == 0 {
		return
	}

	stages := append([]model.FunnelStageSnapshot(nil), funnel...)
	sort.Slice(stages, func(i, j int) bool { return stages[i].StageOrder < stages[j].StageOrder })
	if len(stages) > maxFunnelRows {
		stages = stages[:maxFunnelRows]
	}

	maxCount := 1
	for _, st := range stages {
		if st.Count > maxCount {
			maxCount = st.Count
		}
	}

	const (
		rowH      = 42.0
		barH      = 26.0
		labelW    = 250.0
		countW    = 70.0
		barRadius = 13.0
	)
	top := 408.0
	trackX := pad + labelW
	trackW := width - pad - countW - trackX

	for i, st := range stages {
		y := top + float64(i)*rowH

		dc.SetFontFace(face(fontRegular, 24))
		dc.SetHexColor(colorText)
		dc.DrawStringAnchored(truncate(st.StageName, 20), pad, y+barH/2, 0, 0.5)

		dc.SetHexColor(colorTrack)
		dc.DrawRoundedRectangle(trackX, y, trackW, barH, barRadius)
		dc.Fill()

		fillW := trackW * (float64(st.Count) / float64(maxCount))
		if st.Count > 0 {
			dc.SetHexColor(colorAccent)
			dc.DrawRoundedRectangle(trackX, y, math.Max(fillW, barH), barH, barRadius)
			dc.Fill()
		}

		dc.SetFontFace(face(fontBold, 24))
		dc.SetHexColor(colorText)
		dc.DrawStringAnchored(fmt.Sprintf("%d", st.Count), width-pad, y+barH/2, 1, 0.5)
	}
}

// truncate shortens a label to n runes, adding an ellipsis when cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
