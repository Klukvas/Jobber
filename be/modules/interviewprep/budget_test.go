package interviewprep

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLosslessBudgetReservesAllFollowupsAndEscapedSkips(t *testing.T) {
	snapshot := Snapshot{Job: strings.Repeat("x", 47000)}
	questions := []Question{}
	for i := 0; i < 12; i++ {
		questions = append(questions, Question{ID: fmt.Sprintf("q%d", i), Kind: "technical", Question: strings.Repeat("x", 1500), Purpose: strings.Repeat("x", 400)})
	}
	answers := []Answer{}
	require.Greater(t, remainingBudget(snapshot, questions, answers), 0)
	// Fill capacity using worst JSON escaping as well as multibyte text.
	for i := 0; i < 12; i++ {
		text := strings.Repeat("\x01\\\"😀Ж", 1000)
		candidate := Answer{ID: fmt.Sprintf("00000000-0000-4000-8000-%012d", i), QuestionID: questions[i].ID, Text: text}
		for remainingBudget(snapshot, questions, append(answers, candidate)) < 0 {
			candidate.Text = strings.TrimSuffix(candidate.Text, "\x01\\\"😀Ж")
		}
		answers = append(answers, candidate)
	}
	for i := 0; i < 12; i++ {
		q := Question{ID: fmt.Sprintf("followup:00000000-0000-4000-8000-%012d", i), Kind: "followup", Question: strings.Repeat("\x01", 180), Purpose: strings.Repeat("\x01", 120)}
		raw, _ := json.Marshal(questionContext(q))
		require.LessOrEqual(t, len(raw), 2048)
		questions = append(questions, q)
		answers = append(answers, Answer{ID: fmt.Sprintf("00000000-0000-4000-8001-%012d", i), QuestionID: q.ID, Skipped: true})
		require.GreaterOrEqual(t, remainingBudget(snapshot, questions, answers), 0)
	}
	input := compactInput(snapshot, questions, answers)
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	require.Less(t, len(raw), maxAIInputBytes)
	var decoded AIInput
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, input, decoded)
	require.NotContains(t, string(raw), "feedback")
	require.Len(t, decoded.Questions, 24)
	require.Len(t, decoded.Answers, 24)
}
