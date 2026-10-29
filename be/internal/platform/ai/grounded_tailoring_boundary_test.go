package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestTailoringFinalEncodedFactsNeverTruncated(t *testing.T) {
	source := groundingSource{}
	source.Summary = &struct {
		Content string `json:"content"`
	}{Content: strings.Repeat("S", 4000)}
	for i := 0; i < 14; i++ {
		source.Skills = append(source.Skills, struct{ ID, Name, Level string }{ID: string(rune('a' + i)), Name: strings.Repeat("X", 3980)})
	}
	encode := func() (string, groundingCatalog, int, string) {
		raw, _ := json.Marshal(source)
		catalog, err := resumeGroundingCatalog(string(raw), true)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(catalog)
		view, err := tailoringCatalogJSON(catalog)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw), catalog, len(data), view
	}
	_, _, size, view := encode()
	overhead := len(view) - size
	if overhead <= 0 {
		t.Fatal("fixture does not expose final-view expansion")
	}
	resize := func(target int) {
		_, _, current, _ := encode()
		last := len(source.Skills) - 1
		source.Skills[last].Name = strings.Repeat("X", len(source.Skills[last].Name)+target-current)
	}
	resize(60000 - overhead/2)
	raw, catalog, size, view := encode()
	if size > 60000 || len(view) <= 60000 || validateGroundingInput(catalog, 60000) != nil {
		t.Fatal("invalid expansion boundary fixture")
	}
	calls := 0
	client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		t.Fatal("oversized final facts reached provider")
		return nil, nil
	})
	if _, err := client.TailorResume(t.Context(), "Backend", raw, "en"); err == nil || calls != 0 {
		t.Fatal("oversized final encoded facts were admitted")
	}
	resize(60000 - overhead - 1)
	raw, _, _, view = encode()
	client = newTestClient(func(_ context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		body := p.Messages[0].Content[0].OfText.Text
		marker := "\nresume_facts:\n<document>\n"
		_, facts, ok := strings.Cut(body, marker)
		if !ok || facts != view+"\n</document>" {
			t.Fatal("accepted final JSON was truncated or changed")
		}
		r := mockResponse(`{"summary":null,"experiences":[],"skill_order":null}`)
		r.StopReason = anthropic.StopReasonEndTurn
		return r, nil
	})
	if _, err := client.TailorResume(t.Context(), "Backend", raw, "en"); err != nil || calls != 1 {
		t.Fatal("full boundary source rejected", err)
	}
}
