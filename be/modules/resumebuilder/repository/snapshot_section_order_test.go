package repository

import (
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Hide pgxmock's extra Tx methods to model a real nontransactional pool querier.
type snapshotOrderQuerier struct{ dbQuerier }

func TestSnapshotSectionOrder_DirectCallUsesParentLockTransaction(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM resume_builders WHERE id = \$1 FOR UPDATE`).WithArgs("resume").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("resume"))
	mock.ExpectExec("INSERT INTO resume_section_orders").WithArgs("resume", "skills", 0, true, "main").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	repo := &ResumeBuilderRepository{pool: mock, q: snapshotOrderQuerier{mock}}
	order := &model.SectionOrder{SectionKey: "skills", IsVisible: true}
	before := *order
	require.NoError(t, repo.UpsertSectionOrder(t.Context(), "resume", []*model.SectionOrder{order}))
	assert.Equal(t, before, *order)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSnapshotSectionOrder_LockFailurePreventsSectionWrites(t *testing.T) {
	for _, failure := range []error{pgx.ErrNoRows, errors.New("lock failed")} {
		t.Run(failure.Error(), func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT id FROM resume_builders WHERE id = \$1 FOR UPDATE`).WithArgs("resume").WillReturnError(failure)
			mock.ExpectRollback()
			repo := &ResumeBuilderRepository{pool: mock, q: snapshotOrderQuerier{mock}}
			err = repo.UpsertSectionOrder(t.Context(), "resume", []*model.SectionOrder{{SectionKey: "skills"}})
			if errors.Is(failure, pgx.ErrNoRows) {
				assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
			} else {
				assert.ErrorIs(t, err, failure)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
