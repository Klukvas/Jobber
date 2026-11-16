package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func randomSecret(prefix string, size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	encoded := new(big.Int).SetBytes(data).Text(62)
	width := 43
	if size == 24 {
		width = 32
		maximum := new(big.Int).Exp(big.NewInt(62), big.NewInt(32), nil)
		value, err := rand.Int(rand.Reader, maximum)
		if err != nil {
			return "", err
		}
		encoded = value.Text(62)
	}
	return prefix + strings.Repeat("0", width-len(encoded)) + encoded, nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func validSecret(secret, prefix string) bool {
	if len(secret) != len(prefix)+43 || !strings.HasPrefix(secret, prefix) {
		return false
	}
	for _, char := range secret[len(prefix):] {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

func validVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("-._~", char)) {
			return false
		}
	}
	return true
}

func validChallenge(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == 32 && base64.RawURLEncoding.EncodeToString(decoded) == value
}

func verifyPKCE(verifier, challenge string) bool {
	if !validVerifier(verifier) || !validChallenge(challenge) {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	actual := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(actual), []byte(challenge)) == 1
}

func ValidateRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || strings.Contains(raw, "#") || parsed.Opaque != "" {
		return ErrInvalidRequest
	}
	if parsed.Scheme == "https" {
		if parsed.Port() != "" {
			port, err := strconv.Atoi(parsed.Port())
			if err != nil || port < 1 || port > 65535 {
				return ErrInvalidRequest
			}
		}
		return nil
	}
	if parsed.Scheme != "http" {
		return ErrInvalidRequest
	}
	if !isLoopbackHost(parsed.Hostname()) {
		return ErrInvalidRequest
	}
	// Native clients register loopback callbacks without a port (RFC 8252 §7.3).
	if parsed.Port() != "" {
		port, err := strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return ErrInvalidRequest
		}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// redirectAllowed matches registered callbacks exactly, except that an http
// loopback callback accepts any port: native clients such as Claude Code
// listen on an ephemeral port and register "http://localhost/callback"
// (RFC 8252 §7.3). Scheme, host, path and query must still match exactly.
func redirectAllowed(registered []string, requested string) bool {
	if contains(registered, requested) {
		return true
	}
	target, err := url.Parse(requested)
	if err != nil || target.Scheme != "http" || !isLoopbackHost(target.Hostname()) || ValidateRedirectURI(requested) != nil {
		return false
	}
	for _, candidate := range registered {
		allowed, err := url.Parse(candidate)
		if err == nil && allowed.Scheme == "http" && allowed.Hostname() == target.Hostname() &&
			allowed.EscapedPath() == target.EscapedPath() && allowed.RawQuery == target.RawQuery {
			return true
		}
	}
	return false
}

// ValidateClient returns a normalized copy; callers retain ownership of input.
func ValidateClient(input Client) (Client, error) {
	return validateClient(input, 10)
}

func validateClient(input Client, maxRedirects int) (Client, error) {
	if !utf8.ValidString(input.Name) || utf8.RuneCountInString(input.Name) > 100 || strings.ContainsFunc(input.Name, unicode.IsControl) {
		return Client{}, fmt.Errorf("%w: client_name", ErrInvalidClientMetadata)
	}
	if input.AuthMethod != "" && input.AuthMethod != "none" {
		return Client{}, fmt.Errorf("%w: token_endpoint_auth_method %q", ErrInvalidClientMetadata, input.AuthMethod)
	}
	if len(input.RedirectURIs) < 1 || len(input.RedirectURIs) > maxRedirects {
		return Client{}, fmt.Errorf("%w: %d redirect_uris", ErrInvalidRedirectURI, len(input.RedirectURIs))
	}
	for index, uri := range input.RedirectURIs {
		if err := ValidateRedirectURI(uri); err != nil {
			return Client{}, fmt.Errorf("%w: redirect_uris[%d]", ErrInvalidRedirectURI, index)
		}
	}
	grants, ok := supportedGrants(input.GrantTypes)
	if !ok {
		return Client{}, fmt.Errorf("%w: grant_types", ErrInvalidClientMetadata)
	}
	if !subset(input.ResponseTypes, "code") {
		return Client{}, fmt.Errorf("%w: response_types", ErrInvalidClientMetadata)
	}
	output := input
	output.RedirectURIs = append([]string(nil), input.RedirectURIs...)
	output.GrantTypes = grants
	output.ResponseTypes = append([]string{}, input.ResponseTypes...)
	if input.ResponseTypes == nil {
		output.ResponseTypes = []string{"code"}
	}
	output.AuthMethod = "none"
	return output, nil
}

// Clients may list grants this server lacks (claude.ai lists jwt-bearer);
// RFC 7591 §3.2.1 lets the server register only the ones it supports.
func supportedGrants(requested []string) ([]string, bool) {
	if len(requested) == 0 {
		return []string{"authorization_code", "refresh_token"}, true
	}
	seen := make(map[string]bool)
	grants := []string{}
	for _, grant := range requested {
		if seen[grant] {
			return nil, false
		}
		seen[grant] = true
		if grant == "authorization_code" || grant == "refresh_token" {
			grants = append(grants, grant)
		}
	}
	return grants, len(grants) > 0
}

func subset(values []string, allowed ...string) bool {
	if len(values) > len(allowed) {
		return false
	}
	seen := make(map[string]bool)
	for _, value := range values {
		if seen[value] || !contains(allowed, value) {
			return false
		}
		seen[value] = true
	}
	return true
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
