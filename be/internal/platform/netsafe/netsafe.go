// Package netsafe provides SSRF-hardened URL validation and an HTTP client for
// fetching user-supplied URLs. It is shared so every outbound fetch and every
// write-time URL check enforces the same private-range policy.
package netsafe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// privateCIDRs are ranges that must never be reached via a user-supplied URL.
var privateCIDRs []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("netsafe: invalid CIDR %q: %v", cidr, err))
		}
		privateCIDRs = append(privateCIDRs, block)
	}
}

// IsPrivateIP reports whether an IP falls within a private/reserved range.
func IsPrivateIP(ip net.IP) bool {
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ValidateExternalURL ensures the URL is safe to fetch: https only, non-local
// host, and no DNS resolution to a private/reserved IP.
func ValidateExternalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("only https URLs are allowed, got %q", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" || strings.EqualFold(hostname, "localhost") {
		return fmt.Errorf("hostname %q is not allowed", hostname)
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %q: %w", hostname, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if IsPrivateIP(ip) {
			return fmt.Errorf("hostname %q resolves to private IP %s", hostname, ipStr)
		}
	}

	return nil
}

// SafeClient returns an HTTP client hardened against SSRF. It validates every
// redirect target and, at dial time, re-resolves the host, rejects any private
// IP, and connects to that exact resolved IP — closing the DNS-rebinding window
// between validation and the actual TCP connection.
func SafeClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return ValidateExternalURL(req.URL.String())
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, err)
				}
				for _, ip := range ips {
					if IsPrivateIP(ip.IP) {
						return nil, fmt.Errorf("connection to private IP %s blocked", ip.IP)
					}
				}
				// Dial the validated IP directly so no second resolution can
				// swap in a private address (DNS rebinding).
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
			},
		},
	}
}
