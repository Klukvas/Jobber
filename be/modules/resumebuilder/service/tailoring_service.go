package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/google/uuid"
)

type TailoringAIClient interface {
	TailorResume(context.Context, string, string, string) (*ai.ResumeTailoringSuggestions, error)
}

type TailoringService struct {
	resumes ports.ResumeBuilderRepository
	repo    ports.TailoringRepository
	client  TailoringAIClient
	store   TailoringProposalStore
	limits  LimitChecker
}

func NewTailoringService(resumes ports.ResumeBuilderRepository, repo ports.TailoringRepository, client TailoringAIClient, store TailoringProposalStore, limits LimitChecker) *TailoringService {
	return &TailoringService{resumes: resumes, repo: repo, client: client, store: store, limits: limits}
}

func (s *TailoringService) Generate(ctx context.Context, userID, jobID, resumeID, locale string) (*model.TailoringProposalDTO, error) {
	return s.generate(ctx, userID, jobID, resumeID, locale, uuid.NewString(), true)
}

func (s *TailoringService) generate(ctx context.Context, userID, jobID, resumeID, locale, proposalID string, charge bool) (*model.TailoringProposalDTO, error) {
	job, err := s.repo.GetTailoringJob(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	source, err := s.resumes.GetFullResume(ctx, userID, resumeID)
	if errors.Is(err, model.ErrResumeBuilderNotFound) || errors.Is(err, model.ErrNotOwner) {
		return nil, model.ErrTailoringNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load tailoring source: %w", err)
	}
	return s.generateSnapshot(ctx, userID, jobID, resumeID, locale, proposalID, charge, job, source)
}

func tailoringFacts(job *model.TailoringJob, source *model.FullResumeDTO) ([]byte, error) {
	if strings.TrimSpace(job.Description) == "" {
		return nil, model.ErrTailoringValidation
	}
	visible := visibleTailoringSource(source)
	facts, err := json.Marshal(struct {
		Summary     *model.SummaryDTO      `json:"summary"`
		Experiences []*model.ExperienceDTO `json:"experiences"`
		Skills      []*model.SkillDTO      `json:"skills"`
	}{visible.Summary, visible.Experiences, visible.Skills})
	if err != nil {
		return nil, fmt.Errorf("serialize tailoring facts: %w", err)
	}
	if len(facts) > 60000 || len(source.Experiences) > 50 || len(source.Skills) > 200 {
		return nil, model.ErrTailoringValidation
	}
	return facts, nil
}

func (s *TailoringService) generateSnapshot(ctx context.Context, userID, jobID, resumeID, locale, proposalID string, charge bool, job *model.TailoringJob, source *model.FullResumeDTO) (*model.TailoringProposalDTO, error) {
	if s.client == nil {
		return nil, model.ErrAIUnavailable
	}
	if s.store == nil {
		return nil, model.ErrProposalStoreUnavailable
	}
	facts, err := tailoringFacts(job, source)
	if err != nil {
		return nil, err
	}
	if charge {
		if err := s.limits.CheckLimit(ctx, userID, "ai"); err != nil {
			return nil, err
		}
	}
	ctxAI, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	suggestions, err := s.client.TailorResume(ctxAI, job.Title+"\n"+job.Company+"\n"+job.Description, string(facts), locale)
	if err != nil {
		if errors.Is(err, ai.ErrInvalidTailoringResponse) {
			if charge {
				if usageErr := s.limits.RecordAIUsage(ctx, userID); usageErr != nil {
					slog.Warn("failed to record tailoring AI usage", "error", usageErr)
				}
			}
			return nil, model.ErrAIInvalidResponse
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctxAI.Err(), context.DeadlineExceeded) {
			return nil, model.ErrAITimeout
		}
		slog.Warn("resume tailoring provider failed", "error", err)
		return nil, model.ErrAIRequestFailed
	}
	// A completed provider call counts even if its output is rejected or Redis fails.
	if charge {
		if err := s.limits.RecordAIUsage(ctx, userID); err != nil {
			slog.Warn("failed to record tailoring AI usage", "user_id", userID, "error", err)
		}
	}
	changes, err := validateTailoringSuggestions(source, suggestions)
	if err != nil {
		return nil, err
	}
	proposal := &model.TailoringProposal{
		TailoringProposalDTO: model.TailoringProposalDTO{ProposalID: proposalID, ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
			SourceResume: model.TailoringResource{ID: resumeID, Title: source.Title}, Job: model.TailoringResource{ID: jobID, Title: job.Title}, Changes: changes},
		UserID: userID, Source: source, SourceHash: hashTailoring(source), JobSnapshot: job,
	}
	if err := s.store.Save(ctx, proposal); err != nil {
		return nil, err
	}
	return &proposal.TailoringProposalDTO, nil
}

func (s *TailoringService) Apply(ctx context.Context, userID, jobID, proposalID string, req *model.ApplyTailoringRequest) (*model.ApplyTailoringResult, bool, error) {
	if s.store == nil {
		return nil, false, model.ErrProposalStoreUnavailable
	}
	if req == nil || strings.TrimSpace(req.Title) == "" || len([]rune(strings.TrimSpace(req.Title))) > 255 || len(req.AcceptedChangeIDs) == 0 || len(req.AcceptedChangeIDs) > 52 {
		return nil, false, model.ErrTailoringValidation
	}
	limit, limitErr := resumeBuilderCeiling(ctx, s.limits, userID)
	var result *model.ApplyTailoringResult
	replayed := false
	err := s.repo.RunTailoringTransaction(ctx, userID, func(tx ports.TailoringTransaction) error {
		requestFingerprint := hashApplyRequest(req)
		workflow, durable := tx.(ports.WorkflowTransaction)
		_, durableStore := s.store.(interface{ Durable() bool })
		durable = durable && durableStore
		if durable {
			receipt, err := workflow.GetOperation(ctx, userID, "apply", proposalID)
			if err != nil {
				return err
			}
			if receipt != nil {
				if receipt.JobID != jobID {
					return model.ErrTailoringNotFound
				}
				if receipt.Fingerprint != requestFingerprint {
					return model.ErrOperationConflict
				}
				existing, err := tx.GetByID(ctx, userID, receipt.ObjectID)
				if errors.Is(err, model.ErrResumeBuilderNotFound) {
					return model.ErrOperationDeleted
				}
				if err != nil {
					return err
				}
				result = &model.ApplyTailoringResult{ResumeBuilder: existing.ToDTO(), JobID: jobID}
				replayed = true
				return nil
			}
		}
		var proposal *model.TailoringProposal
		var err error
		if durable {
			operation, loadErr := workflow.GetOperation(ctx, userID, "generate", proposalID)
			if loadErr != nil {
				return loadErr
			}
			if operation == nil || operation.Status != "succeeded" {
				return model.ErrProposalExpired
			}
			if err := json.Unmarshal(operation.Result, &proposal); err != nil {
				return model.ErrProposalStoreUnavailable
			}
			if !time.Now().Before(proposal.ExpiresAt) {
				return model.ErrProposalExpired
			}
		} else {
			proposal, err = s.store.Get(ctx, userID, proposalID)
			if err != nil {
				return err
			}
		}
		if proposal.Job.ID != jobID || proposal.UserID != userID {
			return model.ErrTailoringNotFound
		}
		selected, fingerprint, err := selectTailoringChanges(proposal, req)
		if err != nil {
			return err
		}
		job, err := tx.GetTailoringJob(ctx, userID, jobID)
		if err != nil {
			return err
		}
		existing, err := tx.GetByID(ctx, userID, proposalID)
		if err == nil && existing != nil {
			if proposal.AcceptedFingerprint != fingerprint || job.ResumeBuilderID == nil || *job.ResumeBuilderID != proposalID {
				return model.ErrProposalAlreadyApplied
			}
			result = &model.ApplyTailoringResult{ResumeBuilder: existing.ToDTO(), JobID: jobID}
			replayed = true
			return nil
		}
		if err != nil && !errors.Is(err, model.ErrResumeBuilderNotFound) {
			return err
		}
		if hashTailoring(job) != hashTailoring(proposal.JobSnapshot) {
			return model.ErrJobChanged
		}
		source, err := tx.LockTailoringSource(ctx, userID, proposal.SourceResume.ID)
		if err != nil {
			return err
		}
		if hashTailoring(source) != proposal.SourceHash {
			return model.ErrSourceChanged
		}
		if limitErr != nil {
			return limitErr
		}
		if err := tx.CheckResumeBuilderCapacity(ctx, userID, limit); err != nil {
			return err
		}
		// Persist before committing the copy: a lost response or Redis acknowledgement
		// cannot create a second copy. A rolled-back attempt may choose again.
		proposal.AcceptedFingerprint = fingerprint
		if !durable {
			if err := s.store.Save(ctx, proposal); err != nil {
				return err
			}
		}
		full, err := applyTailoringChanges(source, selected)
		if err != nil {
			return err
		}
		resume := tailoredResumeMetadata(userID, proposalID, strings.TrimSpace(req.Title), source)
		if err := tx.CreateWithID(ctx, resume); err != nil {
			return fmt.Errorf("create tailored resume: %w", err)
		}
		if err := copyResumeSections(ctx, tx, proposalID, full); err != nil {
			return err
		}
		if err := tx.LinkTailoredResume(ctx, userID, jobID, proposalID); err != nil {
			return err
		}
		result = &model.ApplyTailoringResult{ResumeBuilder: resume.ToDTO(), JobID: jobID}
		if durable {
			payload, err := json.Marshal(result)
			if err != nil {
				return err
			}
			return workflow.PutOperation(ctx, &model.WorkflowOperation{UserID: userID, Kind: "apply", Key: proposalID, JobID: jobID, Fingerprint: requestFingerprint, Status: "succeeded", ObjectID: proposalID, Result: payload, CreatedAt: time.Now().UTC()})
		}
		return nil
	})
	return result, replayed, err
}

func hashTailoring(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	} // all callers use JSON-only DTOs
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func selectTailoringChanges(proposal *model.TailoringProposal, req *model.ApplyTailoringRequest) ([]model.TailoringChange, string, error) {
	wanted := make(map[string]bool, len(req.AcceptedChangeIDs))
	for _, id := range req.AcceptedChangeIDs {
		if wanted[id] || id == "" {
			return nil, "", model.ErrTailoringValidation
		}
		wanted[id] = true
	}
	selected := make([]model.TailoringChange, 0, len(wanted))
	for _, change := range proposal.Changes {
		if wanted[change.ID] {
			selected = append(selected, change)
			delete(wanted, change.ID)
		}
	}
	if len(wanted) != 0 {
		return nil, "", model.ErrTailoringValidation
	}
	ids := append([]string(nil), req.AcceptedChangeIDs...)
	sort.Strings(ids)
	return selected, hashTailoring(struct {
		Title string
		IDs   []string
	}{strings.TrimSpace(req.Title), ids}), nil
}

func tailoredResumeMetadata(userID, id, title string, full *model.FullResumeDTO) *model.ResumeBuilder {
	return &model.ResumeBuilder{ID: id, UserID: userID, Title: title, TemplateID: full.TemplateID, FontFamily: full.FontFamily,
		PrimaryColor: full.PrimaryColor, TextColor: full.TextColor, Spacing: full.Spacing, MarginTop: full.MarginTop, MarginBottom: full.MarginBottom,
		MarginLeft: full.MarginLeft, MarginRight: full.MarginRight, LayoutMode: full.LayoutMode, SidebarWidth: full.SidebarWidth, FontSize: full.FontSize, SkillDisplay: full.SkillDisplay}
}

func applyTailoringChanges(source *model.FullResumeDTO, changes []model.TailoringChange) (*model.FullResumeDTO, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var full model.FullResumeDTO
	if err := json.Unmarshal(data, &full); err != nil {
		return nil, err
	}
	for _, change := range changes {
		switch change.Kind {
		case "summary":
			var after string
			if err := json.Unmarshal(change.After, &after); err != nil {
				return nil, model.ErrTailoringValidation
			}
			full.Summary = &model.SummaryDTO{Content: after}
		case "experience":
			var after string
			if err := json.Unmarshal(change.After, &after); err != nil {
				return nil, model.ErrTailoringValidation
			}
			for _, experience := range full.Experiences {
				if experience.ID == change.ExperienceID {
					experience.Description = after
				}
			}
		case "skill_order":
			var refs []model.TailoringSkillRef
			if err := json.Unmarshal(change.After, &refs); err != nil {
				return nil, model.ErrTailoringValidation
			}
			byID := make(map[string]*model.SkillDTO, len(full.Skills))
			for _, skill := range full.Skills {
				byID[skill.ID] = skill
			}
			ordered := make([]*model.SkillDTO, 0, len(refs))
			for i, ref := range refs {
				skill, ok := byID[ref.ID]
				if !ok {
					return nil, model.ErrTailoringValidation
				}
				skill.SortOrder = i
				ordered = append(ordered, skill)
			}
			full.Skills = ordered
		default:
			return nil, model.ErrTailoringValidation
		}
	}
	return &full, nil
}

func hashApplyRequest(req *model.ApplyTailoringRequest) string {
	ids := append([]string(nil), req.AcceptedChangeIDs...)
	sort.Strings(ids)
	return hashTailoring(struct {
		Title string
		IDs   []string
	}{strings.TrimSpace(req.Title), ids})
}
