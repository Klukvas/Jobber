package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/jackc/pgx/v5"
)

// These identifiers are internal constants, never caller-controlled SQL.
var snapshotSectionTables = [...]string{
	"resume_contacts", "resume_summaries", "resume_experiences", "resume_educations",
	"resume_skills", "resume_languages", "resume_certifications", "resume_projects",
	"resume_volunteering", "resume_custom_sections", "resume_section_orders",
}

func (r *snapshotTransaction) LockResume(ctx context.Context, userID, resumeID string) error {
	var id string
	err := r.q.QueryRow(ctx, `SELECT id FROM resume_builders WHERE id = $1 AND user_id = $2 FOR UPDATE`, resumeID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrResumeBuilderNotFound
	}
	if err != nil {
		return fmt.Errorf("lock snapshot resume: %w", err)
	}
	// The parent lock blocks FK inserts; row locks also cover existing section edits.
	for _, table := range snapshotSectionTables {
		if err := r.lockSnapshotSection(ctx, table, resumeID); err != nil {
			return err
		}
	}
	return nil
}

func (r *snapshotTransaction) lockSnapshotSection(ctx context.Context, table, resumeID string) error {
	rows, err := r.q.Query(ctx, `SELECT id FROM `+table+` WHERE resume_builder_id = $1 ORDER BY id FOR UPDATE`, resumeID)
	if err != nil {
		return fmt.Errorf("lock snapshot section: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan locked snapshot section: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read locked snapshot section: %w", err)
	}
	return nil
}

func (r *snapshotTransaction) ClearResumeSections(ctx context.Context, resumeID string) error {
	for _, table := range snapshotSectionTables {
		if _, err := r.q.Exec(ctx, `DELETE FROM `+table+` WHERE resume_builder_id = $1`, resumeID); err != nil {
			return fmt.Errorf("clear snapshot sections: %w", err)
		}
	}
	return nil
}
