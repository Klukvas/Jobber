package contacts

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var emailPattern = regexp.MustCompile(`^[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)+$`)
var telegramPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{4,31}$`)
var linkedinPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

// Normalize accepts only known channel shapes, never arbitrary URL schemes,
// credentials, query strings, ports or encoded path separators.
func Normalize(input Fields) (Fields, error) {
	f := Fields{trimContact(input.Name), trimContact(input.Role), trimContact(input.Email), trimContact(input.Telegram), trimContact(input.LinkedIn), trimContact(input.Note)}
	for _, field := range []struct {
		value     string
		maximum   int
		multiline bool
	}{{f.Name, 120, false}, {f.Role, 120, false}, {f.Email, 254, false}, {f.Telegram, 512, false}, {f.LinkedIn, 512, false}, {f.Note, 2000, true}} {
		if !utf8.ValidString(field.value) || utf8.RuneCountInString(field.value) > field.maximum {
			return Fields{}, ErrInvalid
		}
		for _, r := range field.value {
			if unicode.IsControl(r) && !(field.multiline && (r == '\n' || r == '\r' || r == '\t')) {
				return Fields{}, ErrInvalid
			}
		}
	}
	if f.Name == "" || (f.Email != "" && !emailPattern.MatchString(f.Email)) {
		return Fields{}, ErrInvalid
	}
	if f.Telegram != "" {
		handle := strings.TrimPrefix(f.Telegram, "@")
		for _, prefix := range []string{"https://t.me/", "http://t.me/", "t.me/"} {
			if strings.HasPrefix(f.Telegram, prefix) {
				handle = strings.TrimSuffix(strings.TrimPrefix(f.Telegram, prefix), "/")
				break
			}
		}
		if !telegramPattern.MatchString(handle) {
			return Fields{}, ErrInvalid
		}
		f.Telegram = "https://t.me/" + handle
	}
	if f.LinkedIn != "" {
		slug := ""
		for _, prefix := range []string{"https://www.linkedin.com/in/", "https://linkedin.com/in/", "http://www.linkedin.com/in/", "http://linkedin.com/in/", "www.linkedin.com/in/", "linkedin.com/in/"} {
			if strings.HasPrefix(f.LinkedIn, prefix) {
				slug = strings.TrimSuffix(strings.TrimPrefix(f.LinkedIn, prefix), "/")
				break
			}
		}
		if !linkedinPattern.MatchString(slug) {
			return Fields{}, ErrInvalid
		}
		f.LinkedIn = "https://www.linkedin.com/in/" + slug
	}
	return f, nil
}

func trimContact(value string) string {
	return strings.TrimFunc(value, func(r rune) bool {
		return r == '\u0009' || r == '\u000a' || r == '\u000b' || r == '\u000c' || r == '\u000d' || r == '\u0020' || r == '\u00a0' || r == '\u1680' || (r >= '\u2000' && r <= '\u200a') || r == '\u2028' || r == '\u2029' || r == '\u202f' || r == '\u205f' || r == '\u3000' || r == '\ufeff'
	})
}
