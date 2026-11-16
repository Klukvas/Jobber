package oauth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
const testChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
const callback = "https://claude.ai/api/mcp/auth_callback"

func TestPKCE(t *testing.T) {
	for _, test := range []struct {
		name, verifier, challenge string
		valid                     bool
	}{
		{"RFC7636", testVerifier, testChallenge, true},
		{"wrong", strings.Repeat("A", 43), testChallenge, false},
		{"short", "abc", testChallenge, false},
		{"long", strings.Repeat("a", 129), testChallenge, false},
		{"invalid characters", strings.Repeat("!", 43), testChallenge, false},
		{"padding", testVerifier, testChallenge + "=", false},
		{"noncanonical trailing bits", testVerifier, testChallenge[:42] + "N", false},
	} {
		t.Run(test.name, func(t *testing.T) { require.Equal(t, test.valid, verifyPKCE(test.verifier, test.challenge)) })
	}
}

func TestValidateRedirectURI(t *testing.T) {
	for _, test := range []struct {
		uri   string
		valid bool
	}{
		{callback, true}, {"https://client.test:65536/cb", false}, {"https://chatgpt.com/connector/oauth/id?query=1", true},
		{"http://127.0.0.1:1234/callback", true}, {"http://localhost:1234/cb", true}, {"http://[::1]:1234/cb", true},
		{"http://127.0.0.2:1234/cb", false}, {"http://localhost/cb", true}, {"http://127.0.0.1/callback", true}, {"http://localhost:0/cb", false},
		{"http://localhost:abc/cb", false}, {"http://127.0.0.2/cb", false},
		{"http://localhost:65536/cb", false}, {"http://localhost.evil:123/cb", false}, {"http://example.com:123/cb", false},
		{"https://example.com/cb#frag", false}, {"https://example.com/cb#", false}, {"https://user@example.com/cb", false},
		{"javascript:alert(1)", false}, {"//example.com", false}, {"https://", false}, {"https://example.com/" + strings.Repeat("x", 2048), false},
	} {
		t.Run(test.uri[:min(60, len(test.uri))], func(t *testing.T) { require.Equal(t, test.valid, ValidateRedirectURI(test.uri) == nil) })
	}
}

func TestValidateClient(t *testing.T) {
	valid := Client{Name: "Assistant", RedirectURIs: []string{callback}}
	for _, test := range []struct {
		name  string
		edit  func(Client) Client
		valid bool
	}{
		{"defaults", func(c Client) Client { return c }, true},
		{"public", func(c Client) Client { c.AuthMethod = "none"; return c }, true},
		{"secret", func(c Client) Client { c.AuthMethod = "client_secret_basic"; return c }, false},
		{"empty redirects", func(c Client) Client { c.RedirectURIs = nil; return c }, false},
		{"too many redirects", func(c Client) Client { c.RedirectURIs = make([]string, 11); return c }, false},
		{"bad redirect", func(c Client) Client { c.RedirectURIs = []string{"http://evil.test/cb"}; return c }, false},
		{"long name", func(c Client) Client { c.Name = strings.Repeat("я", 101); return c }, false},
		{"control name", func(c Client) Client { c.Name = "a\nb"; return c }, false},
		{"bad grant", func(c Client) Client { c.GrantTypes = []string{"password"}; return c }, false},
		{"bad response", func(c Client) Client { c.ResponseTypes = []string{"token"}; return c }, false},
		{"duplicate grant", func(c Client) Client { c.GrantTypes = []string{"refresh_token", "refresh_token"}; return c }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateClient(test.edit(valid))
			require.Equal(t, test.valid, err == nil)
		})
	}
	result, err := ValidateClient(valid)
	require.NoError(t, err)
	result.RedirectURIs[0] = "changed"
	require.Equal(t, callback, valid.RedirectURIs[0])
	require.Equal(t, "none", result.AuthMethod)
	require.Equal(t, []string{"authorization_code", "refresh_token"}, result.GrantTypes)
}

func TestRandomSecrets(t *testing.T) {
	for _, prefix := range []string{"", "jbr_oat_", "jbr_ort_"} {
		one, err := randomSecret(prefix, 32)
		require.NoError(t, err)
		two, err := randomSecret(prefix, 32)
		require.NoError(t, err)
		require.True(t, validSecret(one, prefix))
		require.NotEqual(t, one, two)
		require.Len(t, hashSecret(one), 64)
		require.False(t, validSecret(one+"!", prefix))
	}
}

func TestRedirectAllowed(t *testing.T) {
	registered := []string{"http://localhost/callback", "http://127.0.0.1/callback", "https://client.test/cb?x=1"}
	for _, test := range []struct {
		requested string
		allowed   bool
	}{
		{"http://localhost/callback", true},
		{"http://localhost:54321/callback", true},
		{"http://127.0.0.1:8080/callback", true},
		{"https://client.test/cb?x=1", true},
		{"https://client.test:8443/cb?x=1", false},
		{"http://localhost:54321/callback/extra", false},
		{"http://localhost:54321/other", false},
		{"http://localhost:54321/callback?x=1", false},
		{"http://[::1]:54321/callback", false},
		{"https://localhost:54321/callback", false},
		{"http://localhost:0/callback", false},
		{"http://localhost.evil:54321/callback", false},
		{"http://user@localhost:54321/callback", false},
	} {
		t.Run(test.requested, func(t *testing.T) { require.Equal(t, test.allowed, redirectAllowed(registered, test.requested)) })
	}
}
