package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	uploaded "github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const resumeA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const resumeB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
const entryA = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
const entryB = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"

type fakeResumeStore struct {
	mu          sync.Mutex
	resumes     map[string]map[string]any
	owners      map[string]string
	events      []string
	snapshotErr error
	writeErr    error
	readErr     error
	snapshots   map[string]fakeResumeSnapshot
}

type fakeResumeSnapshot struct {
	userID, resumeID, source string
	content                  map[string]any
}

func newFakeResumeStore() *fakeResumeStore {
	store := &fakeResumeStore{
		resumes: make(map[string]map[string]any), owners: map[string]string{resumeA: userA, resumeB: userB},
		events: []string{}, snapshots: make(map[string]fakeResumeSnapshot),
	}
	for id := range store.owners {
		entryID := entryA
		if id == resumeB {
			entryID = entryB
		}
		full := map[string]any{"id": id, "title": "Private " + id, "summary": map[string]any{"content": "Original"}}
		for _, key := range []string{"experiences", "educations", "skills", "languages", "certifications", "projects", "volunteering", "custom_sections"} {
			full[key] = []any{map[string]any{"id": entryID, "name": "Original", "company": "Original"}}
		}
		store.resumes[id] = full
	}
	return store
}

func resumeToolClient(t *testing.T, store *fakeResumeStore, userID string) *sdk.ClientSession {
	t.Helper()
	jobs := newFakeJobs()
	return toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithResumeTools(ResumeTools{
		Builder: store, Uploaded: &fakeUploadedResumes{}, Snapshots: &fakeResumeSnapshots{store: store},
	})), userID)
}

func (f *fakeResumeStore) Get(_ context.Context, userID, id string) (*rb.FullResumeDTO, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, "get:"+userID+":"+id)
	if f.readErr != nil {
		return nil, f.readErr
	}
	if f.owners[id] != userID {
		return nil, rb.ErrResumeBuilderNotFound
	}
	return resumeFakeConvert[rb.FullResumeDTO](f.resumes[id])
}

func (f *fakeResumeStore) List(_ context.Context, userID string) ([]*rb.ResumeBuilderDTO, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]*rb.ResumeBuilderDTO, 0)
	for id, resume := range f.resumes {
		if f.owners[id] == userID {
			item, err := resumeFakeConvert[rb.ResumeBuilderDTO](resume)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	return items, f.readErr
}

func resumeFakeConvert[T any](value any) (*T, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func fakeResumeWrite[T any](f *fakeResumeStore, userID, id, entryID, key string, request any) (*T, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.owners[id] != userID {
		return nil, rb.ErrResumeBuilderNotFound
	}
	f.events = append(f.events, "write:"+key+":"+userID+":"+id)
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	patch, err := resumeFakeConvert[map[string]any](request)
	if err != nil {
		return nil, err
	}
	if key == "contact" || key == "summary" {
		f.resumes[id][key] = *patch
		return resumeFakeConvert[T](*patch)
	}
	entries := f.resumes[id][key].([]any)
	if entryID == "" {
		(*patch)["id"] = uuid.NewString()
		f.resumes[id][key] = append(entries, *patch)
		return resumeFakeConvert[T](*patch)
	}
	for _, item := range entries {
		entry := item.(map[string]any)
		if entry["id"] == entryID {
			for name, value := range *patch {
				entry[name] = value
			}
			return resumeFakeConvert[T](entry)
		}
	}
	return nil, rb.ErrSectionEntryNotFound
}

type fakeUploadedResumes struct{ count int }

func (f *fakeUploadedResumes) List(_ context.Context, userID string, limit, offset int, _, _ string) ([]*uploaded.ResumeDTO, int, error) {
	items := make([]*uploaded.ResumeDTO, 0)
	if userID != userA {
		return items, 0, nil
	}
	count := f.count
	if count == 0 {
		count = 1
	}
	for i := offset; i < count && i < offset+limit; i++ {
		items = append(items, &uploaded.ResumeDTO{ID: uuid.NewString(), Title: "Uploaded", FileURL: ptr("https://private.invalid"), StorageKey: ptr("private-key")})
	}
	return items, count, nil
}

type fakeResumeSnapshots struct{ store *fakeResumeStore }

func (f *fakeResumeSnapshots) SnapshotBeforeWrite(_ context.Context, userID, resumeID, source string) error {
	s := f.store
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "snapshot:"+source+":"+userID+":"+resumeID)
	if s.owners[resumeID] != userID {
		return rb.ErrResumeBuilderNotFound
	}
	if s.snapshotErr != nil {
		return s.snapshotErr
	}
	for _, item := range s.snapshots {
		if item.resumeID == resumeID && item.source == source {
			return nil
		}
	}
	return f.save(userID, resumeID, source)
}

func (f *fakeResumeSnapshots) save(userID, resumeID, source string) error {
	copy, err := resumeFakeConvert[map[string]any](f.store.resumes[resumeID])
	if err != nil {
		return err
	}
	f.store.snapshots[uuid.NewString()] = fakeResumeSnapshot{userID: userID, resumeID: resumeID, source: source, content: *copy}
	return nil
}

func (f *fakeResumeSnapshots) ListSnapshots(_ context.Context, userID, resumeID string) ([]*rb.ResumeSnapshotDTO, error) {
	s := f.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owners[resumeID] != userID {
		return nil, rb.ErrResumeBuilderNotFound
	}
	items := make([]*rb.ResumeSnapshotDTO, 0)
	for id, item := range s.snapshots {
		if item.userID == userID && item.resumeID == resumeID {
			items = append(items, &rb.ResumeSnapshotDTO{ID: id, Source: item.source, CreatedAt: time.Now().UTC()})
		}
	}
	return items, nil
}

func (f *fakeResumeSnapshots) RestoreSnapshot(_ context.Context, userID, id string) (*rb.FullResumeDTO, error) {
	s := f.store
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.snapshots[id]
	if !ok || snapshot.userID != userID {
		return nil, rb.ErrResumeBuilderNotFound
	}
	if s.snapshotErr != nil {
		return nil, s.snapshotErr
	}
	if err := f.save(userID, snapshot.resumeID, "restore"); err != nil {
		return nil, err
	}
	copy, err := resumeFakeConvert[map[string]any](snapshot.content)
	if err != nil {
		return nil, err
	}
	s.resumes[snapshot.resumeID] = *copy
	return resumeFakeConvert[rb.FullResumeDTO](*copy)
}

func requireResumeError(t *testing.T, result *sdk.CallToolResult, text string) {
	t.Helper()
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*sdk.TextContent).Text, text)
}
