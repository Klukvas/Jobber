package email

import (
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
			name:           "ukrainian",
			locale:         "ua",
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
			name:           "ukrainian",
			locale:         "ua",
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
			content := passwordResetEmail("654321", tt.locale)

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
	content := passwordResetEmail("112233", "en")

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
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			content := passwordResetEmail("000000", tt.locale)
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
