package service

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

func replaceResumeContent(ctx context.Context, tx ports.SnapshotTransaction, userID, resumeID string, full *model.FullResumeDTO) error {
	// Identity, ownership and creation time belong to the existing Builder Resume.
	// Update refreshes updated_at; all editable metadata comes from the snapshot.
	metadata := &model.ResumeBuilder{
		ID: resumeID, UserID: userID, Title: full.Title, TemplateID: full.TemplateID,
		FontFamily: full.FontFamily, PrimaryColor: full.PrimaryColor, TextColor: full.TextColor,
		Spacing: full.Spacing, MarginTop: full.MarginTop, MarginBottom: full.MarginBottom,
		MarginLeft: full.MarginLeft, MarginRight: full.MarginRight, LayoutMode: full.LayoutMode,
		SidebarWidth: full.SidebarWidth, FontSize: full.FontSize, SkillDisplay: full.SkillDisplay,
	}
	if err := tx.Update(ctx, metadata); err != nil {
		return fmt.Errorf("restore resume metadata: %w", err)
	}
	if err := tx.ClearResumeSections(ctx, resumeID); err != nil {
		return err
	}
	// Shared with Duplicate: fresh section IDs, with no mutation of the saved DTO.
	return copyResumeSections(ctx, tx, resumeID, full)
}
