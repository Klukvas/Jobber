package interviewprep

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

const maxAIInputBytes = 120000
const maxSourceInputBytes = 48000
const maxQuestionContextBytes = 24000

func questionContext(q Question) QuestionContext {
	return QuestionContext{ID: q.ID, Kind: q.Kind, Question: q.Question, Purpose: q.Purpose}
}
func compactInput(snapshot Snapshot, questions []Question, answers []Answer) AIInput {
	input := AIInput{Snapshot: snapshot}
	for _, q := range questions {
		input.Questions = append(input.Questions, questionContext(q))
	}
	for _, a := range answers {
		input.Answers = append(input.Answers, AIAnswer{ID: a.ID, QuestionID: a.QuestionID, Text: a.Text, Skipped: a.Skipped})
	}
	return input
}

// Reserve escaped JSON bytes for every possible followup and skipped answer.
// Admission uses the exact lossless summary projection, never rune/token estimates.
func remainingBudget(snapshot Snapshot, questions []Question, answers []Answer) int {
	original, followups := 0, 0
	for _, q := range questions {
		if q.Kind == "followup" {
			followups++
		} else {
			original++
		}
	}
	raw, _ := json.Marshal(compactInput(snapshot, questions, answers))
	return maxAIInputBytes - len(raw) - 1024 - (original-followups)*2048 - (2*original-len(answers))*512
}
func loadSnapshot(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, owner, job, version string) (Snapshot, error) {
	var raw []byte
	var snapshot Snapshot
	if err := tx.QueryRow(ctx, `SELECT snapshot FROM interview_versions WHERE user_id=$1 AND job_id=$2 AND id=$3`, owner, job, version).Scan(&raw); err != nil {
		return snapshot, err
	}
	err := json.Unmarshal(raw, &snapshot)
	return snapshot, err
}
