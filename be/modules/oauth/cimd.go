package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}
type Dialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type CIMDFetcher struct{ client *http.Client }

func NewCIMDFetcher(resolver Resolver, dialer Dialer) *CIMDFetcher {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 5 * time.Second}
	}
	transport := &http.Transport{
		// No environment proxy: it would bypass IP validation at the destination.
		DialContext: guardedDial(resolver, dialer), TLSHandshakeTimeout: 5 * time.Second,
		MaxResponseHeaderBytes: 16 << 10, DisableKeepAlives: true,
	}
	return &CIMDFetcher{client: &http.Client{Timeout: 5 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("metadata redirects forbidden") },
	}}
}

func validMetadataURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && len(raw) <= 2048 && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == "" && !strings.Contains(raw, "#") && (parsed.Port() == "" || parsed.Port() == "443") && parsed.Opaque == ""
}

func (f *CIMDFetcher) Fetch(ctx context.Context, raw string) (*Client, error) {
	if !validMetadataURL(raw) {
		return nil, ErrInvalidClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, ErrInvalidClient
	}
	request.Header.Set("Accept", "application/json")
	response, err := f.client.Do(request)
	if err != nil {
		return nil, ErrInvalidClient
	}
	defer response.Body.Close()
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || response.StatusCode != 200 {
		return nil, ErrInvalidClient
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(body) > 64<<10 {
		return nil, ErrInvalidClient
	}
	return parseMetadata(raw, body)
}

func parseMetadata(raw string, body []byte) (*Client, error) {
	var document Client
	if err := json.Unmarshal(body, &document); err != nil || document.ID != raw {
		return nil, ErrInvalidClient
	}
	client, err := validateClient(document, len(document.RedirectURIs))
	if err != nil {
		return nil, ErrInvalidClient
	}
	now := time.Now().UTC()
	client.IsMetadata, client.FetchedAt = true, &now
	return &client, nil
}

func guardedDial(resolver Resolver, dialer Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "443" {
			return nil, ErrInvalidClient
		}
		ips, err := resolver.LookupIPAddr(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, ErrInvalidClient
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, ErrInvalidClient
			}
		}
		// Dial a checked literal, never the hostname, preventing a second DNS lookup.
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		if err != nil {
			return nil, ErrInvalidClient
		}
		actualHost, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		actual := net.ParseIP(actualHost)
		if err != nil || !publicIP(actual) || !actual.Equal(ips[0].IP) {
			if closeErr := conn.Close(); closeErr != nil {
				return nil, errors.Join(ErrInvalidClient, closeErr)
			}
			return nil, ErrInvalidClient
		}
		return conn, nil
	}
}

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2001::/32"),
}

func publicIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}
