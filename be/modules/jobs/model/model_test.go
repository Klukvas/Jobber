package model

import (
	"errors"
	"testing"
	"time"
)

func TestGetErrorCodeAndMessage(t *testing.T) {
	cases := []struct {
		err  error
		code ErrorCode
	}{
		{ErrJobNotFound, CodeJobNotFound},
		{ErrJobTitleRequired, CodeJobTitleRequired},
		{ErrCompanyNotFound, CodeCompanyNotFound},
		{ErrStageTemplateNotFound, CodeStageTemplateNotFound},
		{ErrStageTemplateInUse, CodeStageTemplateInUse},
		{ErrStageTemplateNameExists, CodeStageTemplateNameExists},
		{ErrInvalidMoveTarget, CodeInvalidMoveTarget},
		{ErrJobStageNotFound, CodeJobStageNotFound},
		{ErrInvalidStatus, CodeInvalidStatus},
		{ErrStageNameRequired, CodeStageNameRequired},
		{ErrReorderMismatch, CodeReorderMismatch},
		{ErrBothResumeTypesSet, CodeBothResumeTypesSet},
		{ErrResumeNotFound, CodeResumeNotFound},
		{errors.New("some other error"), CodeInternalError},
	}
	for _, c := range cases {
		if got := GetErrorCode(c.err); got != c.code {
			t.Errorf("GetErrorCode(%v) = %q, want %q", c.err, got, c.code)
		}
		if msg := GetErrorMessage(c.err); msg == "" {
			t.Errorf("GetErrorMessage(%v) returned empty", c.err)
		}
	}
	// Wrapped errors are matched via errors.Is.
	wrapped := errors.Join(errors.New("ctx"), ErrJobNotFound)
	if GetErrorCode(wrapped) != CodeJobNotFound {
		t.Error("GetErrorCode should unwrap via errors.Is")
	}
}

func TestJobToDTO(t *testing.T) {
	now := time.Now().UTC()
	tmplID := "col-1"
	stageID := "hist-1"
	job := &Job{
		ID:                     "job-1",
		Title:                  "Engineer",
		IsFavorite:             true,
		IsArchived:             true,
		AppliedAt:              &now,
		CurrentStageTemplateID: &tmplID,
		CurrentStageID:         &stageID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	dto := job.ToDTO()
	if dto.ID != "job-1" || dto.Title != "Engineer" || !dto.IsFavorite || !dto.IsArchived {
		t.Fatalf("ToDTO scalar fields wrong: %+v", dto)
	}
	if dto.CurrentStageTemplateID == nil || *dto.CurrentStageTemplateID != tmplID {
		t.Error("CurrentStageTemplateID not carried")
	}
	if dto.CurrentStageID == nil || *dto.CurrentStageID != stageID {
		t.Error("CurrentStageID not carried")
	}
	if dto.AppliedAt == nil || !dto.AppliedAt.Equal(now) {
		t.Error("AppliedAt not carried")
	}
	if !dto.LastActivityAt.Equal(now) {
		t.Error("LastActivityAt should default to UpdatedAt")
	}
}

func TestStageTemplateToDTO(t *testing.T) {
	now := time.Now().UTC()
	st := &StageTemplate{ID: "s1", UserID: "u1", Name: "Screening", Order: 2, CreatedAt: now, UpdatedAt: now}
	dto := st.ToDTO()
	if dto.ID != "s1" || dto.Name != "Screening" || dto.Order != 2 || !dto.CreatedAt.Equal(now) {
		t.Errorf("StageTemplate.ToDTO wrong: %+v", dto)
	}
}
