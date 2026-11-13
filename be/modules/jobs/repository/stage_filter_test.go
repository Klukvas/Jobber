package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

// Frozen from the original List query. The matcher deliberately preserves
// whitespace; QueryMatcherEqual alone normalizes it and is not byte-exact.
const originalJobListSQL = `
		WITH stage_activity AS (
			SELECT js.job_id, MAX(js.created_at) as max_created
			FROM job_stages js
			JOIN jobs j ON j.id = js.job_id
			WHERE j.user_id = $1
			GROUP BY js.job_id
		),
		comment_activity AS (
			SELECT cm.job_id, MAX(cm.created_at) as max_created
			FROM comments cm
			JOIN jobs j ON j.id = cm.job_id
			WHERE j.user_id = $1
			GROUP BY cm.job_id
		)
		SELECT
			j.id, j.company_id, j.title, j.source, j.url, j.description,
			j.is_favorite, j.is_archived, j.applied_at,
			j.current_stage_template_id, j.current_stage_id,
			j.created_at, j.updated_at,
			GREATEST(
				j.updated_at,
				COALESCE(sa.max_created, j.updated_at),
				COALESCE(ca.max_created, j.updated_at)
			) as last_activity_at,
			c.name as company_name,
			r.id, r.title,
			rb.id, rb.title,
			st.name as current_stage_name,
			COUNT(*) OVER() as total_count
		FROM jobs j
		LEFT JOIN stage_activity sa ON sa.job_id = j.id
		LEFT JOIN comment_activity ca ON ca.job_id = j.id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN resumes r ON r.id = j.resume_id
		LEFT JOIN resume_builders rb ON rb.id = j.resume_builder_id
		LEFT JOIN stage_templates st ON st.id = j.current_stage_template_id
		WHERE j.user_id = $1%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`

func TestJobRepository_ListWithoutStageUsesOriginalSQL(t *testing.T) {
	for _, test := range []struct {
		name, filter, order string
		opts                ports.ListOptions
		args                []any
	}{
		{"default", " AND j.is_archived = false", "last_activity_at DESC", ports.ListOptions{Limit: 25}, []any{"owner", 25, 0}},
		{"other filters", " AND j.is_archived = true AND (j.title ILIKE $2 OR c.name ILIKE $2) AND j.company_id = $3::uuid", "LOWER(j.title) ASC",
			ports.ListOptions{Limit: 1, Status: "archived", Search: "Engineer", CompanyID: "company", SortBy: "title", SortDir: "asc"}, []any{"owner", "%Engineer%", "company", 1, 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			matcher := pgxmock.QueryMatcherFunc(func(expected, actual string) error {
				if expected != actual {
					return errors.New("SQL differs byte-for-byte from original List query")
				}
				return nil
			})
			mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(matcher))
			require.NoError(t, err)
			t.Cleanup(mock.Close)
			expected := fmt.Sprintf(originalJobListSQL, test.filter, test.order, len(test.args)-1, len(test.args))
			mock.ExpectQuery(expected).WithArgs(test.args...).WillReturnRows(pgxmock.NewRows(listColumns))
			_, _, err = NewJobRepository(mock).List(t.Context(), "owner", &test.opts)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStageFilter(t *testing.T) {
	args := []any{"owner", "%query%"}
	require.Empty(t, stageFilter("", &args))
	require.Equal(t, []any{"owner", "%query%"}, args)
	require.Equal(t, " AND j.current_stage_template_id = $3::uuid", stageFilter("stage-id", &args))
	require.Equal(t, []any{"owner", "%query%", "stage-id"}, args)
}

func TestJobRepository_ListStageFiltersBeforePagination(t *testing.T) {
	repo, mock := newJobRepo(t)
	stageID := "44444444-4444-4444-8444-444444444444"
	now := time.Now().UTC()
	rows := pgxmock.NewRows(listColumns).AddRow("job", nil, "Engineer", nil, nil, nil, false, true, nil, &stageID, nil, now, now, now, nil, nil, nil, nil, nil, nil, 37)
	mock.ExpectQuery(`(?s)j.is_archived = true.*j.title ILIKE \$2.*j.company_id = \$3::uuid.*j.current_stage_template_id = \$4::uuid.*LIMIT \$5 OFFSET \$6`).
		WithArgs("owner", "%Engineer%", "company", stageID, 1, 2).WillReturnRows(rows)
	jobs, total, err := repo.List(context.Background(), "owner", &ports.ListOptions{StageTemplateID: stageID, Status: "archived", Search: "Engineer", CompanyID: "company", Limit: 1, Offset: 2})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, stageID, *jobs[0].CurrentStageTemplateID)
	require.Equal(t, 37, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJobRepository_ListStagePastEndCount(t *testing.T) {
	repo, mock := newJobRepo(t)
	stageID := "44444444-4444-4444-8444-444444444444"
	mock.ExpectQuery(`AND j.current_stage_template_id = \$2::uuid`).
		WithArgs("owner", stageID, 1, 1000).WillReturnRows(pgxmock.NewRows(listColumns))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*WHERE j.user_id = \$1 AND j.is_archived = false AND j.current_stage_template_id = \$2::uuid`).
		WithArgs("owner", stageID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(7))
	jobs, total, err := repo.List(t.Context(), "owner", &ports.ListOptions{StageTemplateID: stageID, Limit: 1, Offset: 1000})
	require.NoError(t, err)
	require.Empty(t, jobs)
	require.Equal(t, 7, total)
	require.NoError(t, mock.ExpectationsWereMet())
}
