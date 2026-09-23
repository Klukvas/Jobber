package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventsAPI models the provider's unprocessed-events list rather than replaying
// canned pages, because the behaviour under test is precisely what the list
// does *while* the sweep is reading it: acknowledging an event removes it, so
// every later page shifts down by one.
type eventsAPI struct {
	mu sync.Mutex
	// unprocessed is the provider's list, oldest first. Acknowledging removes
	// an entry, exactly as the real endpoint does.
	unprocessed []storedEvent
	pageSize    int
	// listed records the event IDs returned by each GET, in order, so a test can
	// assert what the sweep actually saw.
	listed       [][]string
	acknowledged []string
	// ackStatus, when set, is the status every acknowledgement answers with.
	ackStatus int
}

type storedEvent struct {
	id  string
	raw json.RawMessage
}

func (a *eventsAPI) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/events/unprocessed":
			assert.Equal(t, strconv.Itoa(ReconcileWindowDays), r.URL.Query().Get("days"),
				"the sweep must always ask for its documented window")
			a.writePage(w, r)

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/events/"):
			a.acknowledge(w, strings.TrimPrefix(r.URL.Path, "/events/"))

		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (a *eventsAPI) writePage(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, _ = strconv.Atoi(raw)
	}

	size := a.pageSize
	if size <= 0 {
		size = 50
	}
	start := min((page-1)*size, len(a.unprocessed))
	end := min(start+size, len(a.unprocessed))

	events := make([]json.RawMessage, 0, end-start)
	ids := make([]string, 0, end-start)
	for _, stored := range a.unprocessed[start:end] {
		events = append(events, stored.raw)
		ids = append(ids, stored.id)
	}
	a.listed = append(a.listed, ids)

	body := map[string]any{"events": events, "more": end < len(a.unprocessed)}
	if end < len(a.unprocessed) {
		body["nextPage"] = page + 1
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (a *eventsAPI) acknowledge(w http.ResponseWriter, eventID string) {
	a.acknowledged = append(a.acknowledged, eventID)
	if a.ackStatus != 0 {
		w.WriteHeader(a.ackStatus)
		return
	}
	remaining := a.unprocessed[:0]
	for _, stored := range a.unprocessed {
		if stored.id != eventID {
			remaining = append(remaining, stored)
		}
	}
	a.unprocessed = remaining
	_, _ = w.Write([]byte(`{"processed":true}`))
}

func (a *eventsAPI) settled() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.acknowledged...)
}

func (a *eventsAPI) listings() [][]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([][]string(nil), a.listed...)
}

// eventsFrom turns fixture bodies into the provider's unprocessed list.
func eventsFrom(t *testing.T, bodies ...[]byte) []storedEvent {
	t.Helper()
	var stored []storedEvent
	for _, body := range bodies {
		var batch struct {
			Events []json.RawMessage `json:"events"`
		}
		require.NoError(t, json.Unmarshal(body, &batch))
		for _, raw := range batch.Events {
			var envelope struct {
				ID string `json:"id"`
			}
			require.NoError(t, json.Unmarshal(raw, &envelope))
			stored = append(stored, storedEvent{id: envelope.ID, raw: raw})
		}
	}
	return stored
}

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

// withEventID re-stamps a fixture's first event, so several distinct events can
// be built from one recorded payload.
func withEventID(t *testing.T, body []byte, id string) []byte {
	t.Helper()
	return transformFirstEvent(t, body, func(event map[string]any) { event["id"] = id })
}

func TestReconcileAppliesAMissedEventAndSettlesIt(t *testing.T) {
	// The event FastSpring gave up redelivering: nothing in the push channel
	// will ever mention this purchase again, so the pull channel is the only
	// thing between the buyer and a permanently free plan.
	api := &eventsAPI{unprocessed: eventsFrom(t, loadFixture(t, taggedFixture))}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result.Failed, "failures: %v", result.Failed)

	written := repo.lastUpsert(t)
	assert.Equal(t, testUserID, written.UserID)
	assert.Equal(t, PlanEnterprise, written.Plan,
		"a pulled event must grant exactly what the same event delivered by webhook would")

	assert.Equal(t, []string{taggedEventID}, api.settled(),
		"a settled event must be acknowledged so the next sweep does not see it again")
	assert.Empty(t, result.Unacknowledged)
}

func TestReconcileDrainsEveryEventThoughSettlingShiftsTheList(t *testing.T) {
	// The regression this exists for: acknowledging removes events from the
	// provider's list, so the pagination the sweep is walking shifts underneath
	// it — what was page 2 becomes page 1. Advancing the cursor after a page
	// that settled anything therefore steps straight over a page's worth.
	//
	// One event per page makes that unmissable: walking the cursor would settle
	// the first and never see the other two.
	first := loadFixture(t, taggedFixture)
	api := &eventsAPI{
		pageSize: 1,
		unprocessed: eventsFrom(t,
			first,
			withEventID(t, first, "evt-recovered-0002"),
			withEventID(t, first, "evt-recovered-0003"),
		),
	}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result.Failed, "failures: %v", result.Failed)
	assert.ElementsMatch(t,
		[]string{taggedEventID, "evt-recovered-0002", "evt-recovered-0003"},
		api.settled(), "every event must be recovered, not every other one")

	for _, listing := range api.listings() {
		assert.LessOrEqual(t, len(listing), 1)
	}
	assert.Empty(t, api.unprocessed, "the provider's backlog must be empty when the sweep ends")
}

func TestReconcileLeavesAFailedEventUnacknowledged(t *testing.T) {
	// Being unacknowledged is what makes the next sweep retry it, so an event
	// the pipeline could not finish must never be reported as settled.
	body := withoutEventField(t, loadFixture(t, taggedFixture), "account")
	body = withoutEventField(t, body, "tags")

	api := &eventsAPI{unprocessed: eventsFrom(t, body)}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err)
	require.Len(t, result.Failed, 1)
	assert.ErrorIs(t, result.Failed[0].Err, model.ErrSubscriptionNotFound)
	assert.Empty(t, api.settled(), "an unfinished event must stay unprocessed at the provider")
	assert.Empty(t, repo.upserts)
}

func TestReconcileStopsWhenNothingCanBeSettled(t *testing.T) {
	// A page the sweep cannot clear does not shrink the list, so the cursor is
	// the only way forward — and a provider pointing back at the page just read
	// must end the sweep rather than re-reading it to the round cap.
	body := withoutEventField(t, loadFixture(t, taggedFixture), "account")
	body = withoutEventField(t, body, "tags")

	api := &eventsAPI{pageSize: 1, unprocessed: eventsFrom(t, body, withEventID(t, body, "evt-stuck-0002"))}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err)
	assert.Len(t, result.Failed, 2, "the cursor must still advance past events it cannot settle")
	assert.Less(t, result.Listings, maxReconcileRounds,
		"an unsettleable backlog must end the sweep, not run it to the round cap")
}

func TestReconcileAcknowledgesAnEventTheWebhookAlreadyApplied(t *testing.T) {
	// A sweep overlapping a live delivery: the claim recognises the event ID, so
	// nothing is granted twice — but it is still settled, because leaving it
	// unprocessed would have every future sweep pick it up again.
	body := loadFixture(t, taggedFixture)
	api := &eventsAPI{unprocessed: eventsFrom(t, body)}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	_, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))
	require.NoError(t, err)
	writesAfterWebhook := len(repo.upserts)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err)
	assert.Len(t, repo.upserts, writesAfterWebhook, "a duplicate event must not write again")
	assert.Equal(t, []string{taggedEventID}, api.settled())
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, errEventDuplicate)
}

func TestReconcileRefusesToRunWithoutWhatItNeeds(t *testing.T) {
	t.Run("no API credentials", func(t *testing.T) {
		svc := NewSubscriptionService(newRecordingRepo(nil), unconfiguredClient(), testBillingConfig())

		_, err := svc.ReconcileMissedEvents(context.Background())

		assert.ErrorIs(t, err, fastspring.ErrNotConfigured)
	})

	t.Run("no webhook secret to prove an order tag with", func(t *testing.T) {
		// Pulling events with no secret would drag every first purchase in and
		// then refuse it as unprovable, loudly, on every sweep from now on.
		api := &eventsAPI{unprocessed: eventsFrom(t, loadFixture(t, taggedFixture))}
		svc := newReconcileService(t, newRecordingRepo(nil), api)
		svc.cfg.WebhookSecret = ""

		_, err := svc.ReconcileMissedEvents(context.Background())

		assert.ErrorIs(t, err, fastspring.ErrSecretMissing)
		assert.Empty(t, api.listings(), "nothing may be pulled that cannot be applied")
	})
}

func TestReconcileRecordsAnAcknowledgementItCouldNotSend(t *testing.T) {
	// The grant already committed. A failed acknowledgement only means the next
	// sweep sees the event again, where the claim recognises it as a duplicate —
	// but it must be reported, or the counts would claim it was settled.
	api := &eventsAPI{
		unprocessed: eventsFrom(t, loadFixture(t, taggedFixture)),
		ackStatus:   http.StatusInternalServerError,
	}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err, "a purchase that was applied must not be reported as a failed sweep")
	assert.Equal(t, PlanEnterprise, repo.lastUpsert(t).Plan)
	require.Len(t, result.Unacknowledged, 1)
	assert.Equal(t, taggedEventID, result.Unacknowledged[0].EventID)
}

func TestReconcileRecordsEveryEventItGaveUpAcknowledging(t *testing.T) {
	// When the provider's API goes down mid-round the sweep stops calling it,
	// because every remaining call would fail the same way. What it must not do
	// is stop *counting*: an event dropped from the round without being recorded
	// would be reported as settled by a count that never saw it fail.
	//
	// More events than the circuit breaker's failure threshold, deliberately:
	// the breaker only opens on the fourth call, so a shorter list never reaches
	// the branch under test at all.
	first := loadFixture(t, taggedFixture)
	const events = 5
	unprocessed := eventsFrom(t, first)
	for n := 2; n <= events; n++ {
		unprocessed = append(unprocessed,
			eventsFrom(t, withEventID(t, first, fmt.Sprintf("evt-recovered-000%d", n)))...)
	}
	api := &eventsAPI{unprocessed: unprocessed, ackStatus: http.StatusInternalServerError}
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc := newReconcileService(t, repo, api)

	result, err := svc.ReconcileMissedEvents(context.Background())

	require.NoError(t, err)
	assert.Len(t, result.Unacknowledged, events,
		"every event the round gave up on must be accounted for, not just the ones that failed first")
	assert.Less(t, len(api.settled()), events,
		"once the circuit opens the sweep must stop calling an API that is down")
}
