package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventsAPI is a stub of the two Events API calls a sweep makes. It records
// every acknowledgement so a test can assert which events were settled and,
// just as importantly, which were left for the next sweep.
type eventsAPI struct {
	mu sync.Mutex
	// pages is served in order, one per GET.
	pages        []string
	pagesServed  int
	acknowledged []string
	ackStatus    int
}

func (a *eventsAPI) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/events/unprocessed":
			assert.NotEmpty(t, r.URL.Query().Get("days"), "the window is a required parameter")
			if a.pagesServed >= len(a.pages) {
				_, _ = w.Write([]byte(`{"events":[],"more":false,"nextPage":null}`))
				return
			}
			page := a.pages[a.pagesServed]
			a.pagesServed++
			_, _ = w.Write([]byte(page))

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/events/"):
			a.acknowledged = append(a.acknowledged, strings.TrimPrefix(r.URL.Path, "/events/"))
			if a.ackStatus != 0 {
				w.WriteHeader(a.ackStatus)
				return
			}
			_, _ = w.Write([]byte(`{"processed":true}`))

		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (a *eventsAPI) settled() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.acknowledged...)
}

// newReconcileService wires a service to the stub Events API.
func newReconcileService(t *testing.T, repo *recordingRepo, api *eventsAPI) *SubscriptionService {
	t.Helper()
	server := httptest.NewServer(api.handler(t))
	t.Cleanup(server.Close)

	client := fastspring.NewClient(fastspring.Config{
		BaseURL:  server.URL,
		Username: "api-user",
		Password: "api-pass",
	})
	return NewSubscriptionService(repo, client, testBillingConfig())
}

// eventsPage wraps fixture events in the Events API's page envelope.
func eventsPage(t *testing.T, fixtureBody []byte, more bool, nextPage any) string {
	t.Helper()
	var batch struct {
		Events []json.RawMessage `json:"events"`
	}
	require.NoError(t, json.Unmarshal(fixtureBody, &batch))

	page, err := json.Marshal(map[string]any{
		"events":   batch.Events,
		"more":     more,
		"nextPage": nextPage,
	})
	require.NoError(t, err)
	return string(page)
}

func TestReconcileAppliesAMissedEventAndSettlesIt(t *testing.T) {
	// The event FastSpring gave up redelivering: nothing in the push channel
	// will ever mention this purchase again, so the pull channel is the only
	// thing between the buyer and a permanently free plan.
	api := &eventsAPI{pages: []string{eventsPage(t, loadFixture(t, taggedFixture), false, nil)}}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

	require.NoError(t, err)
	assert.Empty(t, result.Failed, "failures: %v", result.Failed)
	assert.Equal(t, 1, result.Pages)

	written := repo.lastUpsert(t)
	assert.Equal(t, testUserID, written.UserID)
	assert.Equal(t, PlanEnterprise, written.Plan,
		"a pulled event must grant exactly what the same event delivered by webhook would")

	assert.Equal(t, []string{taggedEventID}, api.settled(),
		"a settled event must be acknowledged so the next sweep does not see it again")
	assert.Empty(t, result.Unacknowledged)
}

func TestReconcileLeavesAFailedEventUnacknowledged(t *testing.T) {
	// Being unacknowledged is what makes the next sweep retry it, so an event
	// the pipeline could not finish must never be reported as settled.
	body := withoutEventField(t, loadFixture(t, taggedFixture), "account")
	body = withoutEventField(t, body, "tags")

	api := &eventsAPI{pages: []string{eventsPage(t, body, false, nil)}}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

	require.NoError(t, err)
	require.Len(t, result.Failed, 1)
	assert.ErrorIs(t, result.Failed[0].Err, model.ErrSubscriptionNotFound)
	assert.Empty(t, api.settled(), "an unfinished event must stay unprocessed at the provider")
	assert.Empty(t, repo.upserts)
}

func TestReconcileAcknowledgesAnEventTheWebhookAlreadyApplied(t *testing.T) {
	// A sweep overlapping a live delivery: the claim recognises the event ID, so
	// nothing is granted twice — but it is still settled, because leaving it
	// unprocessed would have every future sweep pick it up again.
	body := loadFixture(t, taggedFixture)
	api := &eventsAPI{pages: []string{eventsPage(t, body, false, nil)}}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	_, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))
	require.NoError(t, err)
	writesAfterWebhook := len(repo.upserts)

	result, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

	require.NoError(t, err)
	assert.Len(t, repo.upserts, writesAfterWebhook, "a duplicate event must not write again")
	assert.Equal(t, []string{taggedEventID}, api.settled())
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, errEventDuplicate)
}

func TestReconcileReadsEveryPage(t *testing.T) {
	first := loadFixture(t, taggedFixture)
	second := transformFirstEvent(t, first, func(event map[string]any) {
		event["id"] = "evt-activated-new-account-0002"
	})
	two := 2
	api := &eventsAPI{pages: []string{
		eventsPage(t, first, true, two),
		eventsPage(t, second, false, nil),
	}}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

	require.NoError(t, err)
	assert.Equal(t, 2, result.Pages)
	assert.ElementsMatch(t, []string{taggedEventID, "evt-activated-new-account-0002"}, api.settled())
}

func TestReconcileStopsWhenThePageCursorDoesNotAdvance(t *testing.T) {
	// A provider answering `more: true` while pointing back at the page just
	// read would otherwise loop until the page cap, re-applying the same events
	// and re-acknowledging them every time.
	one := 1
	api := &eventsAPI{pages: []string{
		eventsPage(t, loadFixture(t, taggedFixture), true, one),
	}}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Pages)
}

func TestReconcileRefusesToRunWithoutWhatItNeeds(t *testing.T) {
	t.Run("no API credentials", func(t *testing.T) {
		svc := NewSubscriptionService(newRecordingRepo(nil), unconfiguredClient(), testBillingConfig())

		_, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

		assert.ErrorIs(t, err, fastspring.ErrNotConfigured)
	})

	t.Run("no webhook secret to prove an order tag with", func(t *testing.T) {
		// Pulling events with no secret would drag every first purchase in and
		// then refuse it as unprovable, loudly, on every sweep from now on.
		api := &eventsAPI{pages: []string{eventsPage(t, loadFixture(t, taggedFixture), false, nil)}}
		svc := newReconcileService(t, newRecordingRepo(nil), api)
		svc.cfg.WebhookSecret = ""

		_, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

		assert.ErrorIs(t, err, fastspring.ErrSecretMissing)
		assert.Zero(t, api.pagesServed, "nothing may be pulled that cannot be applied")
	})
}

func TestReconcileRecordsAnAcknowledgementItCouldNotSend(t *testing.T) {
	// The grant already committed. A failed acknowledgement only means the next
	// sweep sees the event again, where the claim recognises it as a duplicate.
	api := &eventsAPI{
		pages:     []string{eventsPage(t, loadFixture(t, taggedFixture), false, nil)},
		ackStatus: http.StatusInternalServerError,
	}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background(), DefaultReconcileDays)

	require.NoError(t, err, "a purchase that was applied must not be reported as a failed sweep")
	assert.Equal(t, PlanEnterprise, repo.lastUpsert(t).Plan)
	require.Len(t, result.Unacknowledged, 1)
	assert.Equal(t, taggedEventID, result.Unacknowledged[0].EventID)
}
