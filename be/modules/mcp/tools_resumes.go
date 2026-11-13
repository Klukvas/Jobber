package mcp

import (
	"context"
	"sort"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerOption adds an optional group of tools without changing existing callers.
type ServerOption func(*tools)

func WithResumeTools(dependencies ResumeTools) ServerOption {
	return func(handlers *tools) { handlers.resumes = &dependencies }
}

func registerResumeTools(server *sdk.Server, handlers *tools) {
	register(server, handlers, "list_resumes", "List Builder Resumes and read-only Uploaded Resume references. Only Builder Resumes can be edited.", true, true, handlers.listResumes)
	register(server, handlers, "get_resume", "Read a full structured Builder Resume and section entry IDs before editing.", true, true, handlers.getResume)
	register(server, handlers, "update_resume_section", "Edit one Builder Resume section. Contact/summary upsert replaces all fields; list updates patch supplied fields. Saves an undo snapshot before writing (at most one per 30-minute window). Use list_resume_snapshots and restore_resume_snapshot to undo. No deletes or reordering.", false, false, handlers.updateResumeSection, resumeSectionInputSchema())
	register(server, handlers, "list_resume_snapshots", "List saved versions of a Builder Resume. Pass a snapshot ID to restore_resume_snapshot to undo edits.", true, true, handlers.listResumeSnapshots)
	register(server, handlers, "restore_resume_snapshot", "Restore a saved Builder Resume version. The restore itself saves the current version first and can also be undone through list_resume_snapshots and restore_resume_snapshot.", false, false, handlers.restoreResumeSnapshot)
}

func (t *tools) listResumes(ctx context.Context, userID string, _ ListResumesInput) (ListResumesOutput, error) {
	builders, err := t.resumes.Builder.List(ctx, userID)
	if err != nil {
		return ListResumesOutput{}, err
	}
	items := make([]ResumeItem, 0, len(builders))
	for _, resume := range builders {
		items = append(items, builderResumeItem(resume))
	}
	// Uploaded Resume service is paginated; do not silently truncate this unified list.
	const pageSize = 100
	for offset := 0; ; offset += pageSize {
		uploaded, total, err := t.resumes.Uploaded.List(ctx, userID, pageSize, offset, "updated_at", "desc")
		if err != nil {
			return ListResumesOutput{}, err
		}
		for _, resume := range uploaded {
			items = append(items, ResumeItem{Kind: "uploaded", ID: resume.ID, Title: resume.Title, UpdatedAt: resume.UpdatedAt})
		}
		if len(uploaded) == 0 || offset+len(uploaded) >= total {
			break
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return ListResumesOutput{Resumes: items}, nil
}

func (t *tools) getResume(ctx context.Context, userID string, input GetResumeInput) (GetResumeOutput, error) {
	id, err := canonicalID(input.ResumeID, "resume_id")
	if err != nil {
		return GetResumeOutput{}, err
	}
	full, err := t.resumes.Builder.Get(ctx, userID, id)
	return GetResumeOutput{Resume: full}, err
}

func (t *tools) listResumeSnapshots(ctx context.Context, userID string, input GetResumeInput) (ListResumeSnapshotsOutput, error) {
	id, err := canonicalID(input.ResumeID, "resume_id")
	if err != nil {
		return ListResumeSnapshotsOutput{}, err
	}
	snapshots, err := t.resumes.Snapshots.ListSnapshots(ctx, userID, id)
	if err != nil {
		return ListResumeSnapshotsOutput{}, err
	}
	items := make([]ResumeSnapshotItem, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items = append(items, ResumeSnapshotItem{ID: snapshot.ID, Source: snapshot.Source, CreatedAt: snapshot.CreatedAt})
	}
	return ListResumeSnapshotsOutput{Snapshots: items}, nil
}

func (t *tools) restoreResumeSnapshot(ctx context.Context, userID string, input RestoreResumeSnapshotInput) (RestoreResumeSnapshotOutput, error) {
	id, err := canonicalID(input.SnapshotID, "snapshot_id")
	if err != nil {
		return RestoreResumeSnapshotOutput{}, err
	}
	full, err := t.resumes.Snapshots.RestoreSnapshot(ctx, userID, id)
	if err != nil {
		return RestoreResumeSnapshotOutput{}, err
	}
	return RestoreResumeSnapshotOutput{
		Resume:       builderResumeItem(full.ResumeBuilderDTO),
		SnapshotNote: "Restore complete. The previous version was saved as a restore snapshot. Use list_resume_snapshots and restore_resume_snapshot to undo this restore.",
	}, nil
}

func builderResumeItem(resume *rb.ResumeBuilderDTO) ResumeItem {
	return ResumeItem{Kind: "builder", ID: resume.ID, Title: resume.Title, UpdatedAt: resume.UpdatedAt}
}
