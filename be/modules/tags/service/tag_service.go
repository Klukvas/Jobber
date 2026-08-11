package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/andreypavlenko/jobber/modules/tags/ports"
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type TagService struct {
	repo ports.TagRepository
}

func NewTagService(repo ports.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) Create(ctx context.Context, userID string, req *model.CreateTagRequest) (*model.TagDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, model.ErrTagNameRequired
	}
	if req.Color != nil && !hexColorRe.MatchString(*req.Color) {
		return nil, model.ErrInvalidColor
	}

	tag := &model.Tag{
		UserID: userID,
		Name:   name,
		Color:  req.Color,
	}
	if err := s.repo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag.ToDTO(), nil
}

func (s *TagService) List(ctx context.Context, userID string) ([]*model.TagDTO, error) {
	tags, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDTOs(tags), nil
}

func (s *TagService) Delete(ctx context.Context, userID, tagID string) error {
	return s.repo.Delete(ctx, userID, tagID)
}

func (s *TagService) Attach(ctx context.Context, userID, tagID string, req *model.AttachTagRequest) error {
	if !isValidEntityType(req.EntityType) {
		return model.ErrInvalidEntityType
	}
	return s.repo.AddRelation(ctx, userID, &model.TagRelation{
		TagID:      tagID,
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
	})
}

func (s *TagService) Detach(ctx context.Context, userID, tagID, entityType, entityID string) error {
	if !isValidEntityType(entityType) {
		return model.ErrInvalidEntityType
	}
	return s.repo.RemoveRelation(ctx, userID, tagID, entityType, entityID)
}

func (s *TagService) ListByEntity(ctx context.Context, userID, entityType, entityID string) ([]*model.TagDTO, error) {
	if !isValidEntityType(entityType) {
		return nil, model.ErrInvalidEntityType
	}
	tags, err := s.repo.ListByEntity(ctx, userID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	return toDTOs(tags), nil
}

func isValidEntityType(entityType string) bool {
	return entityType == model.EntityTypeJob || entityType == model.EntityTypeCompany
}

func toDTOs(tags []*model.Tag) []*model.TagDTO {
	dtos := make([]*model.TagDTO, len(tags))
	for i, tag := range tags {
		dtos[i] = tag.ToDTO()
	}
	return dtos
}
