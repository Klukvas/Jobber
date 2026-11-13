package mcp

import (
	"context"
	"encoding/json"
	"slices"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

type resumeWrite func(context.Context, string, string, string) (map[string]any, error)

type snapshotWriteError struct{ cause error }

func (e snapshotWriteError) Error() string { return "could not save undo snapshot" }
func (e snapshotWriteError) Unwrap() error { return e.cause }

func (t *tools) updateResumeSection(ctx context.Context, userID string, input UpdateResumeSectionInput) (UpdateResumeSectionOutput, error) {
	resumeID, err := canonicalID(input.ResumeID, "resume_id")
	if err != nil {
		return UpdateResumeSectionOutput{}, err
	}
	entryID, err := validateResumeOperation(input)
	if err != nil {
		return UpdateResumeSectionOutput{}, err
	}
	write, err := t.prepareResumeWrite(input)
	if err != nil {
		return UpdateResumeSectionOutput{}, err
	}
	full, err := t.resumes.Builder.Get(ctx, userID, resumeID)
	if err != nil {
		return UpdateResumeSectionOutput{}, err
	}
	if input.Op == "update" && !resumeHasEntry(full, input.Section, entryID) {
		return UpdateResumeSectionOutput{}, rb.ErrSectionEntryNotFound
	}
	if err := t.resumes.Snapshots.SnapshotBeforeWrite(ctx, userID, resumeID, "mcp"); err != nil {
		return UpdateResumeSectionOutput{}, snapshotWriteError{cause: err}
	}
	entry, err := write(ctx, userID, resumeID, entryID)
	if err != nil {
		return UpdateResumeSectionOutput{}, err
	}
	return UpdateResumeSectionOutput{
		Entry:        entry,
		SnapshotNote: "An undo snapshot was saved before editing (one per 30-minute window). Use list_resume_snapshots and restore_resume_snapshot to restore the saved version; that restore is also undoable.",
	}, nil
}

func (t *tools) prepareResumeWrite(input UpdateResumeSectionInput) (resumeWrite, error) {
	builder := t.resumes.Builder
	switch input.Section {
	case "contact":
		return prepareResumeCreate(input.Data, builder.UpsertContact)
	case "summary":
		return prepareResumeCreate(input.Data, builder.UpsertSummary)
	case "experience":
		return prepareResumeEntry(input, builder.CreateExperience, builder.UpdateExperience)
	case "education":
		return prepareResumeEntry(input, builder.CreateEducation, builder.UpdateEducation)
	case "skill":
		return prepareResumeEntry(input, builder.CreateSkill, builder.UpdateSkill)
	case "language":
		return prepareResumeEntry(input, builder.CreateLanguage, builder.UpdateLanguage)
	case "certification":
		return prepareResumeEntry(input, builder.CreateCertification, builder.UpdateCertification)
	case "project":
		return prepareResumeEntry(input, builder.CreateProject, builder.UpdateProject)
	case "volunteering":
		return prepareResumeEntry(input, builder.CreateVolunteering, builder.UpdateVolunteering)
	case "custom":
		return prepareResumeEntry(input, builder.CreateCustomSection, builder.UpdateCustomSection)
	default:
		return nil, validationError("Choose a supported Builder Resume section.")
	}
}

func prepareResumeEntry[Create, Update, Out any](
	input UpdateResumeSectionInput,
	create func(context.Context, string, string, *Create) (*Out, error),
	update func(context.Context, string, string, string, *Update) (*Out, error),
) (resumeWrite, error) {
	if input.Op == "create" {
		return prepareResumeCreate(input.Data, create)
	}
	request, err := decodeResumeData[Update](input.Data)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, userID, resumeID, entryID string) (map[string]any, error) {
		entry, err := update(ctx, userID, resumeID, entryID, request)
		return resumeEntryOutput(entry, err)
	}, nil
}

func prepareResumeCreate[In, Out any](
	data map[string]any,
	write func(context.Context, string, string, *In) (*Out, error),
) (resumeWrite, error) {
	request, err := decodeResumeData[In](data)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, userID, resumeID, _ string) (map[string]any, error) {
		entry, err := write(ctx, userID, resumeID, request)
		return resumeEntryOutput(entry, err)
	}, nil
}

func resumeEntryOutput(entry any, err error) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	output := make(map[string]any)
	if err := json.Unmarshal(encoded, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func resumeHasEntry(full *rb.FullResumeDTO, section, id string) bool {
	switch section {
	case "experience":
		return slices.ContainsFunc(full.Experiences, func(e *rb.ExperienceDTO) bool { return e != nil && e.ID == id })
	case "education":
		return slices.ContainsFunc(full.Educations, func(e *rb.EducationDTO) bool { return e != nil && e.ID == id })
	case "skill":
		return slices.ContainsFunc(full.Skills, func(e *rb.SkillDTO) bool { return e != nil && e.ID == id })
	case "language":
		return slices.ContainsFunc(full.Languages, func(e *rb.LanguageDTO) bool { return e != nil && e.ID == id })
	case "certification":
		return slices.ContainsFunc(full.Certifications, func(e *rb.CertificationDTO) bool { return e != nil && e.ID == id })
	case "project":
		return slices.ContainsFunc(full.Projects, func(e *rb.ProjectDTO) bool { return e != nil && e.ID == id })
	case "volunteering":
		return slices.ContainsFunc(full.Volunteering, func(e *rb.VolunteeringDTO) bool { return e != nil && e.ID == id })
	case "custom":
		return slices.ContainsFunc(full.CustomSections, func(e *rb.CustomSectionDTO) bool { return e != nil && e.ID == id })
	default:
		return false
	}
}
