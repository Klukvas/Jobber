package email

import (
	"net/url"
	"strings"
	"testing"
)

func TestVerificationEmail(t *testing.T) {
	tests := []struct {
		name           string
		locale         string
		wantSubjectHas string
		wantCodeHas    string
		wantHTMLHas    string
	}{
		{
			name:           "english (default)",
			locale:         "en",
			wantSubjectHas: "Verify your email",
			wantCodeHas:    "123456",
			wantHTMLHas:    "Verify your email",
		},
		{
			name:           "russian",
			locale:         "ru",
			wantSubjectHas: "Подтверждение email",
			wantCodeHas:    "123456",
			wantHTMLHas:    "Подтвердите ваш email",
		},
		{
			name:           "ukrainian (legacy ua tag)",
			locale:         "ua",
			wantSubjectHas: "Підтвердження email",
			wantCodeHas:    "123456",
			wantHTMLHas:    "Підтвердіть ваш email",
		},
		{
			// The interface now stores the BCP 47 tag; accounts created before
			// that still carry "ua", and both must reach Ukrainian.
			name:           "ukrainian (bcp 47 uk tag)",
			locale:         "uk",
			wantSubjectHas: "Підтвердження email",
			wantCodeHas:    "123456",
			wantHTMLHas:    "Підтвердіть ваш email",
		},
		{
			name:           "unknown locale falls back to english",
			locale:         "fr",
			wantSubjectHas: "Verify your email",
			wantCodeHas:    "123456",
			wantHTMLHas:    "Verify your email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := verificationEmail("123456", tt.locale)

			if !strings.Contains(content.Subject, tt.wantSubjectHas) {
				t.Errorf("Subject = %q, want containing %q", content.Subject, tt.wantSubjectHas)
			}
			if !strings.Contains(content.HTML, tt.wantCodeHas) {
				t.Errorf("HTML does not contain code %q", tt.wantCodeHas)
			}
			if !strings.Contains(content.HTML, tt.wantHTMLHas) {
				t.Errorf("HTML does not contain %q", tt.wantHTMLHas)
			}
		})
	}
}

func TestPasswordResetEmail(t *testing.T) {
	tests := []struct {
		name           string
		locale         string
		wantSubjectHas string
		wantCodeHas    string
		wantHTMLHas    string
	}{
		{
			name:           "english (default)",
			locale:         "en",
			wantSubjectHas: "Reset your password",
			wantCodeHas:    "654321",
			wantHTMLHas:    "Reset your password",
		},
		{
			name:           "russian",
			locale:         "ru",
			wantSubjectHas: "Сброс пароля",
			wantCodeHas:    "654321",
			wantHTMLHas:    "Сброс пароля",
		},
		{
			name:           "ukrainian (legacy ua tag)",
			locale:         "ua",
			wantSubjectHas: "Скидання пароля",
			wantCodeHas:    "654321",
			wantHTMLHas:    "Скидання пароля",
		},
		{
			name:           "ukrainian (bcp 47 uk tag)",
			locale:         "uk",
			wantSubjectHas: "Скидання пароля",
			wantCodeHas:    "654321",
			wantHTMLHas:    "Скидання пароля",
		},
		{
			name:           "unknown locale falls back to english",
			locale:         "de",
			wantSubjectHas: "Reset your password",
			wantCodeHas:    "654321",
			wantHTMLHas:    "Reset your password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := passwordResetEmail("654321", tt.locale, "")

			if !strings.Contains(content.Subject, tt.wantSubjectHas) {
				t.Errorf("Subject = %q, want containing %q", content.Subject, tt.wantSubjectHas)
			}
			if !strings.Contains(content.HTML, tt.wantCodeHas) {
				t.Errorf("HTML does not contain code %q", tt.wantCodeHas)
			}
			if !strings.Contains(content.HTML, tt.wantHTMLHas) {
				t.Errorf("HTML does not contain %q", tt.wantHTMLHas)
			}
		})
	}
}

func TestVerificationEmail_CodeBlock(t *testing.T) {
	content := verificationEmail("987654", "en")

	if !strings.Contains(content.HTML, "987654") {
		t.Error("verification code not present in HTML")
	}
	if !strings.Contains(content.HTML, "letter-spacing:8px") {
		t.Error("code block should use letter-spacing for readability")
	}
	if !strings.Contains(content.HTML, "Courier") {
		t.Error("code block should use monospace font")
	}
}

func TestPasswordResetEmail_CodeBlock(t *testing.T) {
	content := passwordResetEmail("112233", "en", "")

	if !strings.Contains(content.HTML, "112233") {
		t.Error("reset code not present in HTML")
	}
	if !strings.Contains(content.HTML, "letter-spacing:8px") {
		t.Error("code block should use letter-spacing for readability")
	}
	if !strings.Contains(content.HTML, "Courier") {
		t.Error("code block should use monospace font")
	}
}

func TestVerificationEmail_ExpiryMentioned(t *testing.T) {
	tests := []struct {
		locale  string
		wantHas string
	}{
		{"en", "10 minutes"},
		{"ru", "10 минут"},
		{"ua", "10 хвилин"},
		{"uk", "10 хвилин"},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			content := verificationEmail("000000", tt.locale)
			if !strings.Contains(content.HTML, tt.wantHas) {
				t.Errorf("HTML for locale %q should mention %q expiry", tt.locale, tt.wantHas)
			}
		})
	}
}

func TestPasswordResetEmail_ExpiryMentioned(t *testing.T) {
	tests := []struct {
		locale  string
		wantHas string
	}{
		{"en", "10 minutes"},
		{"ru", "10 минут"},
		{"ua", "10 хвилин"},
		{"uk", "10 хвилин"},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			content := passwordResetEmail("000000", tt.locale, "")
			if !strings.Contains(content.HTML, tt.wantHas) {
				t.Errorf("HTML for locale %q should mention %q expiry", tt.locale, tt.wantHas)
			}
		})
	}
}

func TestCodeBlockHTML(t *testing.T) {
	html := codeBlockHTML("123456")

	if !strings.Contains(html, "123456") {
		t.Error("codeBlockHTML should contain the code")
	}
	if !strings.Contains(html, "font-size:32px") {
		t.Error("codeBlockHTML should render code in large font")
	}
	if !strings.Contains(html, "monospace") {
		t.Error("codeBlockHTML should use monospace font family")
	}
}

// The reset page takes both halves of the credential from the URL —
// `/reset-password?email=…&code=…` — and refuses to render a form without
// either. The email used to carry only the code, so the one link a customer
// has in front of them could not be followed: they had to find the app, find
// the forgot-password screen and retype six digits from another window.
func TestResetPasswordURL(t *testing.T) {
	t.Run("builds the deep link the reset page expects", func(t *testing.T) {
		got := resetPasswordURL("https://jobber-app.com", "alex@example.com", "654321")

		parsed, err := url.Parse(got)
		if err != nil {
			t.Fatalf("not a URL: %v", err)
		}
		if parsed.Scheme != "https" || parsed.Host != "jobber-app.com" {
			t.Errorf("wrong origin: %s", got)
		}
		if parsed.Path != "/reset-password" {
			t.Errorf("wrong path: %s", parsed.Path)
		}
		if q := parsed.Query(); q.Get("email") != "alex@example.com" || q.Get("code") != "654321" {
			t.Errorf("wrong query: %s", parsed.RawQuery)
		}
	})

	t.Run("keeps a base URL's own path prefix", func(t *testing.T) {
		got := resetPasswordURL("https://example.test/app/", "a@b.test", "111111")

		if !strings.HasPrefix(got, "https://example.test/app/reset-password?") {
			t.Errorf("path prefix lost: %s", got)
		}
	})

	t.Run("percent-encodes the address rather than pasting it in", func(t *testing.T) {
		got := resetPasswordURL("https://jobber-app.com", "a+b@example.com", "654321")

		parsed, err := url.Parse(got)
		if err != nil {
			t.Fatalf("not a URL: %v", err)
		}
		if parsed.Query().Get("email") != "a+b@example.com" {
			t.Errorf("address did not survive the round trip: %s", parsed.RawQuery)
		}
	})

	// A link is only worth sending if it goes where it says. Anything this
	// cannot vouch for produces no link at all, and the email falls back to the
	// code on its own — which is what it has always been able to do.
	t.Run("refuses a base URL it cannot vouch for", func(t *testing.T) {
		for _, base := range []string{
			"",
			"not a url at all",
			"/relative/only",
			"javascript:alert(1)",
			"ftp://files.example.test",
			"https://",
		} {
			if got := resetPasswordURL(base, "a@b.test", "654321"); got != "" {
				t.Errorf("base %q produced %q, want no link", base, got)
			}
		}
	})

	t.Run("refuses to build a link with a missing half", func(t *testing.T) {
		if got := resetPasswordURL("https://jobber-app.com", "", "654321"); got != "" {
			t.Errorf("no address, still a link: %q", got)
		}
		if got := resetPasswordURL("https://jobber-app.com", "a@b.test", ""); got != "" {
			t.Errorf("no code, still a link: %q", got)
		}
	})

	// A stray query or fragment on the configured origin must not survive into
	// a link whose whole meaning is its query string.
	t.Run("replaces anything the base URL already carried", func(t *testing.T) {
		got := resetPasswordURL("https://jobber-app.com/?utm=x#frag", "a@b.test", "654321")

		parsed, err := url.Parse(got)
		if err != nil {
			t.Fatalf("not a URL: %v", err)
		}
		if parsed.Fragment != "" || parsed.Query().Get("utm") != "" {
			t.Errorf("carried the base's own query/fragment: %s", got)
		}
	})
}

func TestPasswordResetEmail_DeepLink(t *testing.T) {
	link := resetPasswordURL("https://jobber-app.com", "alex@example.com", "654321")

	for _, locale := range []string{"en", "ru", "ua", "uk"} {
		content := passwordResetEmail("654321", locale, link)

		// `&` inside an HTML attribute has to be written as an entity, or the
		// mail client's parser truncates the query at the second parameter and
		// the link arrives without the code.
		if !strings.Contains(content.HTML, `href="https://jobber-app.com/reset-password?code=654321&amp;email=alex%40example.com"`) {
			t.Errorf("%s: reset link missing or unescaped", locale)
		}
		// The code stays visible either way: a mail client that strips links,
		// a customer reading on one device and typing on another.
		if !strings.Contains(content.HTML, "654321") {
			t.Errorf("%s: code no longer shown", locale)
		}
	}
}

func TestPasswordResetEmail_WithoutDeepLink(t *testing.T) {
	for _, locale := range []string{"en", "ru", "ua", "uk"} {
		content := passwordResetEmail("654321", locale, "")

		if strings.Contains(content.HTML, "reset-password?") {
			t.Errorf("%s: emitted a link with nothing to point at", locale)
		}
		if strings.Contains(content.HTML, `href=""`) {
			t.Errorf("%s: emitted an empty href", locale)
		}
		if !strings.Contains(content.HTML, "654321") {
			t.Errorf("%s: code missing from the fallback", locale)
		}
	}
}
