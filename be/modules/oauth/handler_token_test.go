package oauth

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRevokeTokenOnlyAndPublicClientAuthentication(t *testing.T) {
	for _, test := range []struct {
		name, clientID, authorization string
		status                        int
	}{
		{"token only", "", "", 200},
		{"explicit client", "client", "", 200},
		{"Basic is rejected", "client", "Basic Y2xpZW50OnNlY3JldA==", 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			router, _, store, _ := testHandler(t)
			token := "jbr_ort_" + strings.Repeat("A", 43)
			values := url.Values{"token": {token}, "token_type_hint": {"refresh_token"}}
			if test.clientID != "" {
				values.Set("client_id", test.clientID)
			}
			request := httptest.NewRequest("POST", "/api/oauth/revoke", strings.NewReader(values.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Authorization", test.authorization)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code)
			if test.status == 200 {
				require.Equal(t, hashSecret(token), store.revokedHash)
				require.Equal(t, test.clientID, store.revokedClient)
			} else {
				require.JSONEq(t, `{"error":"invalid_client"}`, response.Body.String())
				require.Empty(t, store.revokedHash)
			}
		})
	}
}
