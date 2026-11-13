package service

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	seen := make(map[string]bool)
	format := regexp.MustCompile(`^jbr_pat_[0-9a-zA-Z]{43}$`)
	for range 1000 {
		token, err := generateToken()
		if err != nil {
			t.Fatal(err)
		}
		if !format.MatchString(token) || !validToken(token) {
			t.Fatal("invalid generated token format")
		}
		if seen[token] {
			t.Fatal("duplicate random token")
		}
		seen[token] = true
		if len(hashToken(token)) != 64 {
			t.Fatal("invalid hash length")
		}
	}
}

func TestValidToken(t *testing.T) {
	tests := []struct{ name, input string }{
		{"empty", ""}, {"short", "jbr_pat_a"},
		{"wrong prefix", "other__" + strings.Repeat("a", 43)},
		{"symbol", TokenPrefix + strings.Repeat("a", 42) + "_"},
		{"too long", TokenPrefix + strings.Repeat("a", 44)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if validToken(test.input) {
				t.Fatal("accepted malformed token")
			}
		})
	}
}
