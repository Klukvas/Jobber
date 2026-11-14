package contacts

import (
	"context"

	"github.com/andreypavlenko/jobber/internal/platform/ids"
)

type Store interface {
	List(context.Context, string, string) ([]Contact, error)
	Create(context.Context, string, string, Fields) (*Contact, error)
	Update(context.Context, string, string, string, Fields) (*Contact, error)
	Delete(context.Context, string, string, string) error
}
type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

// scopeIDs returns the user and job ids in canonical form, or ErrNotFound when
// either is not a uuid. Only the canonical form is forwarded: Postgres rejects
// some spellings uuid.Parse accepts, such as `urn:uuid:…`.
func scopeIDs(user, job string) (userID, jobID string, err error) {
	scope, ok := ids.CanonicalUUIDs(user, job)
	if !ok {
		return "", "", ErrNotFound
	}
	return scope[0], scope[1], nil
}
func (s *Service) List(ctx context.Context, user, job string) ([]Contact, error) {
	userID, jobID, err := scopeIDs(user, job)
	if err != nil {
		return nil, err
	}
	return s.store.List(ctx, userID, jobID)
}
func (s *Service) Save(ctx context.Context, user, job, id string, input Fields) (*Contact, error) {
	userID, jobID, err := scopeIDs(user, job)
	if err != nil {
		return nil, err
	}
	contactID := ""
	if id != "" {
		var ok bool
		if contactID, ok = ids.CanonicalUUID(id); !ok {
			return nil, ErrNotFound
		}
	}
	f, err := Normalize(input)
	if err != nil {
		return nil, err
	}
	if contactID == "" {
		return s.store.Create(ctx, userID, jobID, f)
	}
	return s.store.Update(ctx, userID, jobID, contactID, f)
}
func (s *Service) Delete(ctx context.Context, user, job, id string) error {
	userID, jobID, err := scopeIDs(user, job)
	if err != nil {
		return err
	}
	contactID, ok := ids.CanonicalUUID(id)
	if !ok {
		return ErrNotFound
	}
	return s.store.Delete(ctx, userID, jobID, contactID)
}
