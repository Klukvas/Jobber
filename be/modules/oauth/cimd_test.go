package oauth

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type resolverFunc func(context.Context, string) ([]net.IPAddr, error)

func (f resolverFunc) LookupIPAddr(c context.Context, h string) ([]net.IPAddr, error) { return f(c, h) }

type dialerFunc func(context.Context, string, string) (net.Conn, error)

func (f dialerFunc) DialContext(c context.Context, n, a string) (net.Conn, error) { return f(c, n, a) }

type connStub struct {
	net.Conn
	remote net.Addr
	closed bool
}

func (c *connStub) RemoteAddr() net.Addr { return c.remote }
func (c *connStub) Close() error         { c.closed = true; return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCIMDValidation(t *testing.T) {
	for _, test := range []struct {
		url   string
		valid bool
	}{
		{"https://client.example/metadata", true}, {"https://client.example:443/metadata", true},
		{"http://client.example/meta", false}, {"https://client.example:8443/meta", false},
		{"https://user:pass@client.example/meta", false}, {"https://client.example/meta#", false},
	} {
		require.Equal(t, test.valid, validMetadataURL(test.url), test.url)
	}
	for _, body := range []string{
		`{}`, `{"client_id":"https://other.test","redirect_uris":["` + callback + `"]}`,
		`{"client_id":"https://client.test/meta","redirect_uris":["http://evil.test/cb"]}`,
		`{"client_id":"https://client.test/meta","redirect_uris":["` + callback + `"],"token_endpoint_auth_method":"client_secret_post"}`,
	} {
		_, err := parseMetadata("https://client.test/meta", []byte(body))
		require.Error(t, err)
	}
	client, err := parseMetadata("https://client.test/meta", []byte(`{"client_id":"https://client.test/meta","redirect_uris":["`+callback+`"]}`))
	require.NoError(t, err)
	require.True(t, client.IsMetadata)
	require.NotNil(t, client.FetchedAt)
}

func TestGuardedDial(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.1.1", "172.16.1.1", "192.168.1.1", "169.254.169.254", "100.64.1.2", "224.0.0.1", "0.0.0.0", "::", "::1", "fe80::1", "fd00::1", "ff02::1", "::ffff:127.0.0.1", "64:ff9b::7f00:1"} {
		t.Run(ip, func(t *testing.T) {
			resolver := resolverFunc(func(context.Context, string) ([]net.IPAddr, error) { return []net.IPAddr{{IP: net.ParseIP(ip)}}, nil })
			dialer := dialerFunc(func(context.Context, string, string) (net.Conn, error) { t.Fatal("unsafe dial"); return nil, nil })
			_, err := guardedDial(resolver, dialer)(t.Context(), "tcp", "client.test:443")
			require.Error(t, err)
		})
	}
	for _, remote := range []string{"93.184.216.34", "127.0.0.1", "8.8.8.8"} {
		t.Run("actual "+remote, func(t *testing.T) {
			resolver := resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
				return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
			})
			conn := &connStub{remote: &net.TCPAddr{IP: net.ParseIP(remote), Port: 443}}
			dialer := dialerFunc(func(_ context.Context, _, address string) (net.Conn, error) {
				require.Equal(t, "93.184.216.34:443", address)
				return conn, nil
			})
			_, err := guardedDial(resolver, dialer)(t.Context(), "tcp", "client.test:443")
			require.Equal(t, remote == "93.184.216.34", err == nil)
			require.Equal(t, remote != "93.184.216.34", conn.closed)
		})
	}
}

func TestCIMDFetchResponseBounds(t *testing.T) {
	valid := `{"client_id":"https://client.test/meta","redirect_uris":["` + callback + `"]}`
	for _, test := range []struct {
		name, media, body string
		status            int
		valid             bool
	}{
		{"valid", "application/json", valid, 200, true}, {"media", "text/html", valid, 200, false},
		{"large", "application/json", strings.Repeat(" ", 64<<10) + valid, 200, false}, {"status", "application/json", valid, 500, false},
		{"trailing JSON", "application/json", valid + `{}`, 200, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fetcher := NewCIMDFetcher(nil, nil)
			fetcher.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Header: http.Header{"Content-Type": []string{test.media}}, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})
			_, err := fetcher.Fetch(t.Context(), "https://client.test/meta")
			require.Equal(t, test.valid, err == nil)
		})
	}
	require.Error(t, NewCIMDFetcher(nil, nil).client.CheckRedirect(nil, nil))
}
