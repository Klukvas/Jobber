package contacts

import (
	"context"

	"github.com/google/uuid"
)

type Store interface {
	List(context.Context, string, string) ([]Contact, error)
	Create(context.Context, string, string, Fields) (*Contact, error)
	Update(context.Context, string, string, string, Fields) (*Contact, error)
	Delete(context.Context, string, string, string) error
}
type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }
func validIDs(ids ...string) bool {
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return false
		}
	}
	return true
}
func (s *Service) List(ctx context.Context, user, job string) ([]Contact, error) {
	if !validIDs(user, job) {
		return nil, ErrNotFound
	}
	return s.store.List(ctx, user, job)
}
func (s *Service) Save(ctx context.Context, user, job, id string, input Fields) (*Contact, error) {
	if !validIDs(user, job) || (id != "" && !validIDs(id)) {
		return nil, ErrNotFound
	}
	f, err := Normalize(input)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return s.store.Create(ctx, user, job, f)
	}
	return s.store.Update(ctx, user, job, id, f)
}
func (s *Service) Delete(ctx context.Context, user, job, id string) error {
	if !validIDs(user, job, id) {
		return ErrNotFound
	}
	return s.store.Delete(ctx, user, job, id)
}
