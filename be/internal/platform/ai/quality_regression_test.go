package ai

import (
	"context"
	"reflect"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestQualityRegressionLettersRejectUnsupportedCandidateClaims(t *testing.T) {
	client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		response := mockResponse(`"greeting":"Dear Hiring Manager,","paragraphs":["I am applying to Acme.","I have C systems programming experience and mentored engineers.","I would welcome an interview."],"closing":"Sincerely,"}`)
		response.StopReason = anthropic.StopReasonEndTurn
		return response, nil
	})
	if _, err := client.GenerateCoverLetter(t.Context(), "Acme", "", "", "C systems programming and mentoring required", "Python backend engineer. Reduced costs by 3 million dollars."); err == nil {
		t.Fatal("letter accepted job qualifications as candidate facts")
	}
}

func TestQualityRegressionTailoringCompletesKnownSkillRanking(t *testing.T) {
	facts := `{"summary":{"content":"Backend engineer."},"experiences":[],"skills":[{"id":"a","name":"Python"},{"id":"b","name":"Go"},{"id":"c","name":"SQL"}]}`
	client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		response := mockResponse(`{"summary":null,"experiences":[],"skill_order":{"skill_ids":["b"],"reason":"Prioritize the relevant existing skill.","evidence":[{"section":"skills","entry_id":"b","quote":"Go"}]}}`)
		response.StopReason = anthropic.StopReasonEndTurn
		return response, nil
	})
	result, err := client.TailorResume(t.Context(), "Go backend role", facts, "en")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.SkillOrder.SkillIDs, []string{"b", "a", "c"}) {
		t.Fatal("partial known ranking did not retain remaining source skills")
	}
}
