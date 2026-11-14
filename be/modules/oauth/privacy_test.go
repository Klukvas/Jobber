package oauth

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type privacyTransport struct{ events []*sentry.Event }

func (*privacyTransport) Configure(sentry.ClientOptions)        {}
func (*privacyTransport) Flush(time.Duration) bool              { return true }
func (*privacyTransport) FlushWithContext(context.Context) bool { return true }
func (*privacyTransport) Close()                                {}
func (p *privacyTransport) SendEvent(event *sentry.Event)       { p.events = append(p.events, event) }

func TestOAuthRequestPrivacy(t *testing.T) {
	for _, path := range []string{"/api/oauth/token", "/api/v1/oauth/authorize", "/api/v1/oauth/authorize/preview"} {
		t.Run(path, func(t *testing.T) {
			transport := &privacyTransport{}
			client, err := sentry.NewClient(sentry.ClientOptions{Dsn: "https://public@example.invalid/1", Transport: transport, DisableTelemetryBuffer: true})
			require.NoError(t, err)
			defer client.Close()
			hub := sentry.NewHub(client, sentry.NewScope())
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Request = c.Request.WithContext(sentry.SetHubOnContext(c.Request.Context(), hub))
				c.Next()
			}, sentrygin.New(sentrygin.Options{}), Privacy())
			router.POST(path, func(c *gin.Context) {
				sentry.GetHubFromContext(c.Request.Context()).CaptureMessage("oauth test event")
				c.Status(200)
			})
			request := httptest.NewRequest("POST", path+"?state=secret-state", strings.NewReader("code=secret-code&code_verifier=secret-verifier"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Authorization", "Bearer secret-access")
			router.ServeHTTP(httptest.NewRecorder(), request)
			require.Len(t, transport.events, 1)
			require.Nil(t, transport.events[0].Request)
			encoded, err := json.Marshal(transport.events)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "secret-")
		})
	}
}
