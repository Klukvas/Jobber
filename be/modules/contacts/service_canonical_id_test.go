package contacts

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/stretchr/testify/require"
)

// recordingStore remembers the ids each call received.
type recordingStore struct{ calls [][]string }

func (s *recordingStore) record(ids ...string) { s.calls = append(s.calls, ids) }
func (s *recordingStore) List(_ context.Context, user, job string) ([]Contact, error) {
	s.record(user, job)
	return nil, nil
}
func (s *recordingStore) Create(_ context.Context, user, job string, _ Fields) (*Contact, error) {
	s.record(user, job)
	return &Contact{}, nil
}
func (s *recordingStore) Update(_ context.Context, user, job, id string, _ Fields) (*Contact, error) {
	s.record(user, job, id)
	return &Contact{}, nil
}
func (s *recordingStore) Delete(_ context.Context, user, job, id string) error {
	s.record(user, job, id)
	return nil
}

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so
// every id must reach the store in canonical form or the request is a 500.
func TestServiceForwardsCanonicalIDs(t *testing.T) {
	const (
		user    = "11111111-1111-4111-8111-111111111111"
		job     = "22222222-2222-4222-8222-222222222222"
		contact = "33333333-3333-4333-8333-333333333333"
	)
	userForms, jobForms, contactForms := idstest.NonCanonicalForms(user), idstest.NonCanonicalForms(job), idstest.NonCanonicalForms(contact)

	for name := range jobForms {
		t.Run(name, func(t *testing.T) {
			store := &recordingStore{}
			svc := NewService(store)
			ctx := context.Background()
			u, j, c := userForms[name], jobForms[name], contactForms[name]

			_, err := svc.List(ctx, u, j)
			require.NoError(t, err)
			_, err = svc.Save(ctx, u, j, "", Fields{Name: "Ada"})
			require.NoError(t, err)
			_, err = svc.Save(ctx, u, j, c, Fields{Name: "Ada"})
			require.NoError(t, err)
			require.NoError(t, svc.Delete(ctx, u, j, c))

			require.Equal(t, [][]string{
				{user, job},
				{user, job},
				{user, job, contact},
				{user, job, contact},
			}, store.calls)
		})
	}
}

// An invalid contact id still wins over invalid fields, as before.
func TestServiceSaveRejectsInvalidIDBeforeFields(t *testing.T) {
	svc := NewService(&recordingStore{})
	_, err := svc.Save(context.Background(),
		"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222",
		"not-a-uuid", Fields{Name: " "})
	require.ErrorIs(t, err, ErrNotFound)
}
