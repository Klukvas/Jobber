package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest records what the service sent to the provider.
type capturedRequest struct {
	Method string
	Path   string
	Query  string
	Auth   bool
	Body   map[string]any
}

// newStubProvider starts a fake FastSpring API returning the given handler, and
// a service wired to it.
func newStubProvider(t *testing.T, repo *MockSubscriptionRepository, handler http.HandlerFunc) (*SubscriptionService, *[]capturedRequest) {
	t.Helper()
	var captured []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry := capturedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery}
		_, _, entry.Auth = r.BasicAuth()
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&entry.Body)
		}
		captured = append(captured, entry)
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client := fastspring.NewClient(fastspring.Config{
		BaseURL:  server.URL,
		Username: "test-user",
		Password: "test-pass",
	})
	return NewSubscriptionService(repo, client, testBillingConfig()), &captured
}

// sessionResponseBody is a current-Sessions-API 201 body for the fixtures.
//
// It still carries `checkoutUrls`, exactly as the provider sends it — that is
// the point: the service must ignore it. Nothing may put a hosted checkout URL
// back in front of a buyer.
const sessionResponseBody = `{
	"id":"sEsS10nIdTest001",
	"expires":"2026-07-01T00:00:00Z",
	"checkoutStatus":["READY_FOR_CHECKOUT"],
	"customer":{"accountId":"acctTest001","externalAccountId":"jobber-test"},
	"checkoutUrls":{"webcheckoutUrl":"https://fluxlab.test.onfastspring.com/checkout/sEsS10nIdTest001"}
}`

func TestCreateCheckoutSession(t *testing.T) {
	t.Run("creates a server-side session and links the account", func(t *testing.T) {
		var linkedUser, linkedAccount string
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(_ context.Context, userID, accountID string) error {
				linkedUser, linkedAccount = userID, accountID
				return nil
			},
		}
		svc, captured := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(sessionResponseBody))
		})

		session, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.NoError(t, err)
		assert.Equal(t, "sEsS10nIdTest001", session.SessionID,
			"the popup opens the session id; that is the whole contract with the browser")
		assert.Equal(t, "2026-07-01T00:00:00Z", session.ExpiresAt)

		require.Len(t, *captured, 1)
		req := (*captured)[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/v2/checkouts/"+testCheckoutPath+"/sessions", req.Path,
			"checkout sessions must go through the current Sessions API")
		assert.True(t, req.Auth, "the provider call must be authenticated")

		lineItems, ok := req.Body["cart"].(map[string]any)["lineItems"].([]any)
		require.True(t, ok)
		require.Len(t, lineItems, 1)
		assert.Equal(t, testProProductPath, lineItems[0].(map[string]any)["productPath"])
		assert.Equal(t, float64(1), lineItems[0].(map[string]any)["quantity"])

		// The user identity travels server-to-server only.
		customer := req.Body["customer"].(map[string]any)
		assert.Equal(t, accountLookupKey(testUserID), customer["externalAccountId"])
		assert.Equal(t, "buyer@example.com", customer["billToContact"].(map[string]any)["email"])
		assert.Equal(t, "Test", customer["billToContact"].(map[string]any)["firstName"])
		assert.Equal(t, "Buyer", customer["billToContact"].(map[string]any)["lastName"])
		assert.NotContains(t, customer["billToContact"], "country",
			"country is a top-level session field, not a contact field — sending it here would do nothing")

		// Diagnostic only — nothing resolves a user from it.
		assert.Equal(t, testUserID, req.Body["orderTags"].(map[string]any)[userIDTagKey])

		assert.Equal(t, testUserID, linkedUser)
		assert.Equal(t, "acctTest001", linkedAccount)
	})

	t.Run("sends the store mode explicitly", func(t *testing.T) {
		tests := []struct {
			environment string
			wantLive    bool
		}{
			{environment: EnvironmentTest, wantLive: false},
			{environment: EnvironmentLive, wantLive: true},
		}

		for _, tc := range tests {
			t.Run(tc.environment, func(t *testing.T) {
				svc, captured := newStubProvider(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(sessionResponseBody))
				})
				svc.cfg.Environment = tc.environment

				_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

				require.NoError(t, err)
				require.Len(t, *captured, 1)
				assert.Equal(t, tc.wantLive, (*captured)[0].Body["live"],
					"a test deployment must never open a live checkout, or the reverse")
			})
		}
	})

	t.Run("localises the checkout from the buyer record", func(t *testing.T) {
		// FastSpring takes a two-letter language code, and its checkout language
		// set has no Ukrainian — "ua"/"uk" therefore render in Russian, the
		// store's default language for Ukraine (ADR-0002 flags that default as a
		// dashboard setting to confirm in test mode).
		tests := map[string]string{
			"en": "en",
			"ru": "ru",
			"ua": "ru",
			"uk": "ru",
			"RU": "ru",
			"":   "en",
			"kl": "en",
		}

		for locale, want := range tests {
			t.Run("locale "+locale, func(t *testing.T) {
				repo := &MockSubscriptionRepository{
					GetUserContactFunc: func(context.Context, string) (*model.UserContact, error) {
						return &model.UserContact{Email: "buyer@example.com", Name: "Test Buyer", Locale: locale}, nil
					},
				}
				svc, captured := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(sessionResponseBody))
				})

				_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

				require.NoError(t, err)
				require.Len(t, *captured, 1)
				assert.Equal(t, want, (*captured)[0].Body["locale"],
					"the checkout locale must be a 2-letter language code, not a regional tag")
			})
		}
	})

	t.Run("refuses a session that is not ready for checkout", func(t *testing.T) {
		var linked bool
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error {
				linked = true
				return nil
			},
		}
		svc, _ := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"sess-1","checkoutStatus":["PRODUCTS_REQUIRED"],
				"customer":{"accountId":"acct-1"}}`))
		})

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not ready for checkout")
		assert.Contains(t, err.Error(), "PRODUCTS_REQUIRED",
			"the error must name the statuses the provider actually reported")
		assert.False(t, linked, "an unusable session must not link a billing account")
	})

	t.Run("refuses a session whose status list has no READY_FOR_CHECKOUT", func(t *testing.T) {
		// `checkoutStatus` is a list, so a session can report several statuses at
		// once. Readiness is membership: none of these grants a checkout.
		tests := []struct {
			name     string
			statuses string
		}{
			{name: "products required", statuses: `["PRODUCTS_REQUIRED"]`},
			{name: "already concluded", statuses: `["CONCLUDED"]`},
			{name: "several, none ready", statuses: `["PRODUCTS_REQUIRED","CONCLUDED"]`},
			{name: "empty list", statuses: `[]`},
			{name: "null", statuses: `null`},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc, _ := newStubProvider(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":"sess-1","checkoutStatus":%s,
						"customer":{"accountId":"acct-1"}}`, tc.statuses)
				})

				_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "not ready for checkout")
			})
		}
	})

	t.Run("hands the browser a session id and no URL at all", func(t *testing.T) {
		// The popup takes the session id. A URL in this DTO would be a way back
		// into the full-page redirect this integration deliberately dropped —
		// and a provider-supplied address the browser follows blindly.
		svc, _ := newStubProvider(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(sessionResponseBody))
		})

		session, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.NoError(t, err)
		encoded, err := json.Marshal(session)
		require.NoError(t, err)

		var fields map[string]any
		require.NoError(t, json.Unmarshal(encoded, &fields))
		assert.Equal(t, []string{"expires_at", "session_id"}, sortedKeys(fields),
			"the checkout DTO carries a session id and an expiry, nothing else")
		assert.NotContains(t, string(encoded), "onfastspring.com",
			"no provider URL may reach the browser through this DTO")
		assert.NotContains(t, string(encoded), testProProductPath,
			"catalog product paths stay server-side")
	})

	t.Run("omits an expiry the provider did not send in ISO 8601", func(t *testing.T) {
		svc, _ := newStubProvider(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"sess-1","expires":1751328000000,"checkoutStatus":["READY_FOR_CHECKOUT"],
				"customer":{"accountId":"acct-1"}}`))
		})

		session, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.NoError(t, err)
		assert.Empty(t, session.ExpiresAt, "expiry is advisory; a bad value must not fail a working checkout")
		assert.Equal(t, "sess-1", session.SessionID)
	})

	t.Run("rejects a checkout path that could reach another endpoint", func(t *testing.T) {
		svc, captured := newStubProvider(t, &MockSubscriptionRepository{}, func(http.ResponseWriter, *http.Request) {
			t.Fatal("provider must not be called with an unsafe checkout path")
		})
		svc.cfg.CheckoutPath = "jobberstore/../accounts/acct-1"

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.ErrorIs(t, err, fastspring.ErrInvalidCheckoutPath)
		assert.Empty(t, *captured)
	})

	t.Run("reports an unconfigured checkout path", func(t *testing.T) {
		svc := newTestService(&MockSubscriptionRepository{})
		svc.cfg.CheckoutPath = ""

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.Error(t, err)
	})

	t.Run("rejects a plan with no configured product", func(t *testing.T) {
		svc := newTestService(&MockSubscriptionRepository{})

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, "platinum")

		require.ErrorIs(t, err, model.ErrUnknownPlan)
	})

	t.Run("fails safely without credentials", func(t *testing.T) {
		svc := NewSubscriptionService(&MockSubscriptionRepository{}, unconfiguredClient(), testBillingConfig())

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.ErrorIs(t, err, fastspring.ErrNotConfigured)
		assert.NotContains(t, err.Error(), "test-pass")
	})

	t.Run("reports provider rejection without leaking credentials", func(t *testing.T) {
		svc, _ := newStubProvider(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"account":"invalid"}}`))
		})

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.NotContains(t, err.Error(), "test-pass")
	})
}

func TestChangePlan(t *testing.T) {
	repoWithSubscription := func() *MockSubscriptionRepository {
		return &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return activeProSubscription(), nil
			},
		}
	}

	t.Run("switches the provider subscription to the new product", func(t *testing.T) {
		svc, captured := newStubProvider(t, repoWithSubscription(), func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"subscriptions":[{"subscription":"` + fixtureSubscriptionID + `","action":"subscription.update","result":"success"}]}`))
		})

		require.NoError(t, svc.ChangePlan(context.Background(), testUserID, PlanEnterprise))

		require.Len(t, *captured, 1)
		req := (*captured)[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/subscriptions", req.Path)

		items := req.Body["subscriptions"].([]any)
		require.Len(t, items, 1)
		item := items[0].(map[string]any)
		assert.Equal(t, fixtureSubscriptionID, item["subscription"])
		assert.Equal(t, testEnterpriseProductPath, item["product"])
		assert.Equal(t, true, item["prorate"])
	})

	t.Run("surfaces a per-item error inside a 200 response", func(t *testing.T) {
		svc, _ := newStubProvider(t, repoWithSubscription(), func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"subscriptions":[{"subscription":"x","action":"subscription.update","result":"error","error":{"product":"invalid"}}]}`))
		})

		err := svc.ChangePlan(context.Background(), testUserID, PlanEnterprise)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "subscription.update")
	})

	t.Run("rejects an unknown plan before calling the provider", func(t *testing.T) {
		svc, captured := newStubProvider(t, repoWithSubscription(), func(http.ResponseWriter, *http.Request) {
			t.Fatal("provider must not be called for an unknown plan")
		})

		require.ErrorIs(t, svc.ChangePlan(context.Background(), testUserID, "platinum"), model.ErrUnknownPlan)
		assert.Empty(t, *captured)
	})

	t.Run("reports when there is nothing to change", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return linkedFreeSubscription(), nil
			},
		}
		svc, _ := newStubProvider(t, repo, func(http.ResponseWriter, *http.Request) {
			t.Fatal("provider must not be called without a subscription")
		})

		require.ErrorIs(t, svc.ChangePlan(context.Background(), testUserID, PlanEnterprise), model.ErrNoActiveSubscription)
	})
}

func TestCancelSubscription(t *testing.T) {
	repo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
			return activeProSubscription(), nil
		},
	}

	t.Run("cancels at the end of the billing period", func(t *testing.T) {
		svc, captured := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"subscriptions":[{"subscription":"` + fixtureSubscriptionID + `","action":"subscription.cancel","result":"success"}]}`))
		})

		require.NoError(t, svc.CancelSubscription(context.Background(), testUserID))

		require.Len(t, *captured, 1)
		req := (*captured)[0]
		assert.Equal(t, http.MethodDelete, req.Method)
		assert.Equal(t, "/subscriptions/"+fixtureSubscriptionID, req.Path)
		assert.Equal(t, "billingPeriod=1", req.Query, "billingPeriod=1 keeps access until the period ends")
	})

	t.Run("treats an empty result list as a failure", func(t *testing.T) {
		svc, _ := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"subscriptions":[]}`))
		})

		require.Error(t, svc.CancelSubscription(context.Background(), testUserID))
	})
}

func TestCreatePortalSession(t *testing.T) {
	t.Run("returns an authenticated portal URL on the subscriptions tab", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return activeProSubscription(), nil
			},
		}
		svc, captured := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"accounts":[{"account":"` + fixtureAccountID + `","result":"success","url":"https://jobber.test.onfastspring.com/account/abc/def"}]}`))
		})

		url, err := svc.CreatePortalSession(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, "https://jobber.test.onfastspring.com/account/abc/def#/subscriptions", url)
		require.Len(t, *captured, 1)
		assert.Equal(t, "/accounts/"+fixtureAccountID+"/authenticate", (*captured)[0].Path)
	})

	t.Run("reports when the user has no billing account", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return &model.Subscription{UserID: testUserID, Plan: PlanFree, Status: StatusFree}, nil
			},
		}
		svc, _ := newStubProvider(t, repo, func(http.ResponseWriter, *http.Request) {
			t.Fatal("provider must not be called without an account")
		})

		_, err := svc.CreatePortalSession(context.Background(), testUserID)

		require.ErrorIs(t, err, model.ErrNoActiveSubscription)
	})
}

func TestGetCheckoutConfig(t *testing.T) {
	t.Run("advertises the configured plans without credentials", func(t *testing.T) {
		config := newTestService(&MockSubscriptionRepository{}).GetCheckoutConfig()

		assert.Equal(t, Provider, config.Provider)
		assert.Equal(t, EnvironmentTest, config.Environment)
		assert.Equal(t, []string{PlanPro, PlanEnterprise}, config.Plans)

		encoded, err := json.Marshal(config)
		require.NoError(t, err)
		for _, secret := range []string{"test-user", "test-pass", testWebhookSecret} {
			assert.NotContains(t, string(encoded), secret, "checkout config must not carry credentials")
		}
		for _, productPath := range []string{testProProductPath, testEnterpriseProductPath} {
			assert.NotContains(t, string(encoded), productPath,
				"catalog product paths are server-side; the browser only names plans")
		}
	})

	t.Run("derives the popup storefront from the checkout path and store mode", func(t *testing.T) {
		tests := []struct {
			environment string
			want        string
		}{
			{environment: EnvironmentTest, want: testTestStorefront},
			{environment: EnvironmentLive, want: testLiveStorefront},
		}

		for _, tc := range tests {
			t.Run(tc.environment, func(t *testing.T) {
				svc := newTestService(&MockSubscriptionRepository{})
				svc.cfg.Environment = tc.environment

				config := svc.GetCheckoutConfig()

				assert.Equal(t, tc.want, config.Storefront)
				assert.Equal(t, tc.environment, config.Environment,
					"the environment travels with the storefront so the browser can cross-check it")
			})
		}
	})

	t.Run("advertises no storefront for a checkout the popup cannot open", func(t *testing.T) {
		// Startup validation makes this unreachable while payments are on. If it
		// ever is reached, the frontend must read "not openable" rather than be
		// handed a storefront guessed from a bad path.
		for _, path := range []string{"", "fluxlab/jobber-checkout", "fluxlab/../accounts"} {
			t.Run(path, func(t *testing.T) {
				svc := newTestService(&MockSubscriptionRepository{})
				svc.cfg.CheckoutPath = path

				assert.Empty(t, svc.GetCheckoutConfig().Storefront)
			})
		}
	})

	t.Run("omits a plan with no product path", func(t *testing.T) {
		svc := newTestService(&MockSubscriptionRepository{})
		svc.cfg.EnterpriseProductPath = ""

		assert.Equal(t, []string{PlanPro}, svc.GetCheckoutConfig().Plans)
	})
}

func TestAccountLookupKeyRoundTrip(t *testing.T) {
	key := accountLookupKey(testUserID)

	assert.True(t, strings.HasPrefix(key, accountLookupPrefix))
	assert.GreaterOrEqual(t, len(key), 4, "FastSpring requires at least 4 characters")

	decoded, ok := userIDFromLookupKey(key)
	require.True(t, ok)
	assert.Equal(t, testUserID, decoded)

	for _, foreign := range []string{"", "someone-elses-key", accountLookupPrefix + "short", accountLookupPrefix + strings.Repeat("z", 32)} {
		_, ok := userIDFromLookupKey(foreign)
		assert.False(t, ok, "must not decode a key Jobber did not create: %q", foreign)
	}
}

func TestCreateCheckoutSessionRefusesASecondSubscription(t *testing.T) {
	// A user's row carries a single external_subscription_id. A second checkout
	// would overwrite it, leaving the first subscription billing at the provider
	// with nothing in Jobber pointing at it — uncancellable from either side.
	cancelAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	refused := []struct {
		name string
		sub  *model.Subscription
	}{
		{
			name: "active",
			sub:  activeProSubscription(),
		},
		{
			name: "past_due still bills during dunning",
			sub: &model.Subscription{
				UserID: testUserID, ExternalSubscriptionID: ptr(fixtureSubscriptionID),
				Status: StatusPastDue, Plan: PlanPro,
			},
		},
		{
			name: "paused is suspended, not ended",
			sub: &model.Subscription{
				UserID: testUserID, ExternalSubscriptionID: ptr(fixtureSubscriptionID),
				Status: StatusPaused, Plan: PlanPro,
			},
		},
		{
			name: "cancellation scheduled — still active until the period ends",
			sub: &model.Subscription{
				UserID: testUserID, ExternalSubscriptionID: ptr(fixtureSubscriptionID),
				Status: StatusActive, Plan: PlanPro, CancelAt: &cancelAt,
			},
		},
	}

	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
					return tc.sub, nil
				},
				GetUserContactFunc: func(context.Context, string) (*model.UserContact, error) {
					t.Fatal("a refused checkout must not even load the buyer contact")
					return nil, nil
				},
			}
			svc, captured := newStubProvider(t, repo, func(http.ResponseWriter, *http.Request) {
				t.Fatal("provider must not be called for a user who already subscribes")
			})

			_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanEnterprise)

			require.ErrorIs(t, err, model.ErrAlreadySubscribed)
			assert.Empty(t, *captured)
		})
	}

	allowed := []struct {
		name string
		sub  *model.Subscription
		err  error
	}{
		{
			name: "cancelled — the provider ended it, so buying again is the only way back",
			sub: &model.Subscription{
				UserID: testUserID, ExternalSubscriptionID: ptr(fixtureSubscriptionID),
				Status: StatusCancelled, Plan: PlanFree,
			},
		},
		{
			name: "linked account but no subscription — checkout was started, never completed",
			sub:  linkedFreeSubscription(),
		},
		{
			name: "empty subscription ID is not a subscription",
			sub: &model.Subscription{
				UserID: testUserID, ExternalSubscriptionID: ptr(""),
				Status: StatusFree, Plan: PlanFree,
			},
		},
		{
			name: "no row at all — the user has never checked out",
			err:  model.ErrSubscriptionNotFound,
		},
	}

	for _, tc := range allowed {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
					return tc.sub, tc.err
				},
			}
			svc, captured := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(sessionResponseBody))
			})

			session, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

			require.NoError(t, err)
			assert.NotEmpty(t, session.SessionID)
			assert.Len(t, *captured, 1, "a first checkout must reach the provider")
		})
	}

	t.Run("an unreadable subscription fails the checkout rather than bypassing the guard", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return nil, errors.New("connection reset by peer")
			},
		}
		svc, captured := newStubProvider(t, repo, func(http.ResponseWriter, *http.Request) {
			t.Fatal("provider must not be called when the guard could not run")
		})

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrAlreadySubscribed,
			"a database failure must not masquerade as a business rule")
		assert.Contains(t, err.Error(), "connection reset by peer")
		assert.Empty(t, *captured)
	})
}

func TestCreatePortalSessionRefusesAnUnsafeURL(t *testing.T) {
	// The browser is navigated straight to whatever this returns, so a scheme
	// that is not HTTPS must never leave the service. The host is deliberately
	// not pinned — a FastSpring storefront may run on a custom domain, so there
	// is no documented host contract to whitelist against (see ADR-0002).
	unsafe := map[string]string{
		"javascript":      "javascript:alert(1)",
		"plain http":      "http://store.onfastspring.com/account/abc",
		"scheme relative": "//evil.example.com/account",
		"relative path":   "/account/abc",
		"data":            "data:text/html,<script>alert(1)</script>",
		"empty":           "",
		"whitespace":      "   ",
		"no host":         "https:///account/abc",
	}

	for name, portalURL := range unsafe {
		t.Run(name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
					return activeProSubscription(), nil
				},
			}
			svc, _ := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(w, `{"accounts":[{"account":%q,"result":"success","url":%q}]}`,
					fixtureAccountID, portalURL)
			})

			_, err := svc.CreatePortalSession(context.Background(), testUserID)

			require.Error(t, err, "a subscriber must never be navigated to %q", portalURL)
		})
	}

	t.Run("a custom storefront domain is accepted", func(t *testing.T) {
		// Pinning the host to *.onfastspring.com would break this the moment a
		// custom domain is configured in the dashboard.
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return activeProSubscription(), nil
			},
		}
		svc, _ := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"accounts":[{"account":"` + fixtureAccountID +
				`","result":"success","url":"https://billing.jobber-app.com/account/abc"}]}`))
		})

		url, err := svc.CreatePortalSession(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, "https://billing.jobber-app.com/account/abc#/subscriptions", url)
	})

	t.Run("an existing portal route is preserved", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return activeProSubscription(), nil
			},
		}
		svc, _ := newStubProvider(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"accounts":[{"account":"` + fixtureAccountID +
				`","result":"success","url":"https://billing.jobber-app.com/account/abc?locale=ru#/orders"}]}`))
		})

		url, err := svc.CreatePortalSession(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, "https://billing.jobber-app.com/account/abc?locale=ru#/orders", url)
	})
}

// sortedKeys lists a decoded JSON object's field names, so a DTO's exact wire
// shape can be asserted rather than only the fields that happen to be present.
func sortedKeys(fields map[string]any) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
