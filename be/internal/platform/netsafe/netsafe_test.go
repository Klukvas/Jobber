package netsafe

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip      string
		private bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"169.254.169.254", true}, // link-local / cloud metadata
		{"::1", true},
		{"fc00::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"172.32.0.1", false}, // just outside 172.16/12
		{"93.184.216.34", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("bad test IP %q", c.ip)
		}
		if got := IsPrivateIP(ip); got != c.private {
			t.Errorf("IsPrivateIP(%s) = %v, want %v", c.ip, got, c.private)
		}
	}
}

func TestValidateExternalURL(t *testing.T) {
	// All cases are network-free: rejections happen before DNS, and IP-literal
	// hosts are resolved by the stdlib without a real DNS query.
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"non-https scheme", "http://example.com/f.pdf", true},
		{"ftp scheme", "ftp://example.com/f.pdf", true},
		{"empty host", "https:///f.pdf", true},
		{"localhost", "https://localhost/f.pdf", true},
		{"localhost mixed case", "https://LOCALHOST/f.pdf", true},
		{"private IP literal", "https://10.0.0.1/f.pdf", true},
		{"link-local metadata IP", "https://169.254.169.254/latest", true},
		{"loopback IP literal", "https://127.0.0.1/f.pdf", true},
		{"unparseable", "://not-a-url", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateExternalURL(c.url)
			if (err != nil) != c.wantErr {
				t.Errorf("ValidateExternalURL(%q) err=%v, wantErr=%v", c.url, err, c.wantErr)
			}
		})
	}
}

func TestSafeClientConfigured(t *testing.T) {
	c := SafeClient()
	if c.Timeout == 0 {
		t.Error("SafeClient must set a timeout")
	}
	if c.CheckRedirect == nil {
		t.Error("SafeClient must set a CheckRedirect hook")
	}
	// Redirect to a private host must be refused.
	req, _ := http.NewRequest(http.MethodGet, "https://10.0.0.1/x", nil)
	if err := c.CheckRedirect(req, nil); err == nil {
		t.Error("CheckRedirect must reject a private-IP redirect target")
	}
	// A non-https redirect target is refused without needing the network.
	badReq, _ := http.NewRequest(http.MethodGet, "http://example.com/x", nil)
	if err := c.CheckRedirect(badReq, nil); err == nil || !strings.Contains(err.Error(), "https") {
		t.Errorf("CheckRedirect must reject non-https redirect, got %v", err)
	}
}

func TestSafeClientDialRejectsPrivateIP(t *testing.T) {
	tr, ok := SafeClient().Transport.(*http.Transport)
	if !ok || tr.DialContext == nil {
		t.Fatal("SafeClient must configure a Transport with a custom DialContext")
	}
	// An IP-literal host is resolved without a real DNS query, so this asserts
	// the dial-time private-IP guard (the DNS-rebinding fix) without the network.
	_, err := tr.DialContext(context.Background(), "tcp", "10.0.0.1:443")
	if err == nil || !strings.Contains(err.Error(), "private IP") {
		t.Errorf("DialContext must block a private IP, got %v", err)
	}
}
