package contacts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	good := Fields{Name: "  Ірина 😀 ", Role: " Recruiter ", Email: "a?subject=x@example.com", Telegram: "http://t.me/Recruiter_1/", LinkedIn: "linkedin.com/in/recruiter-1/", Note: " <script>text</script>\nprivate "}
	got, err := Normalize(good)
	require.NoError(t, err)
	require.Equal(t, "Ірина 😀", got.Name)
	require.Equal(t, "https://t.me/Recruiter_1", got.Telegram)
	require.Equal(t, "https://www.linkedin.com/in/recruiter-1", got.LinkedIn)
	require.Equal(t, "<script>text</script>\nprivate", got.Note)
	_, err = Normalize(Fields{Name: strings.Repeat("😀", 120), Note: strings.Repeat("😀", 2000)})
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		fields Fields
	}{
		{"empty", Fields{Name: " \n"}}, {"name too long", Fields{Name: strings.Repeat("😀", 121)}}, {"role", Fields{Name: "N", Role: strings.Repeat("r", 121)}}, {"note", Fields{Name: "N", Note: strings.Repeat("n", 2001)}},
		{"email newline", Fields{Name: "N", Email: "a@example.com\r\nBcc:x@example.com"}}, {"email scheme", Fields{Name: "N", Email: "mailto:a@example.com"}}, {"control", Fields{Name: "N\x00"}},
	} {
		t.Run(tc.name, func(t *testing.T) { _, err := Normalize(tc.fields); require.ErrorIs(t, err, ErrInvalid) })
	}
	for _, value := range []string{"javascript:alert(1)", "https://t.me.evil/abcde", "https://evil@t.me/abcde", "https://t.me:443/abcde", "t.me/abcde?x=1", "t.me/%61bcde", "t.me/@abcde", "@abcde/", "abcd", "12345"} {
		t.Run("telegram "+value, func(t *testing.T) {
			_, err := Normalize(Fields{Name: "N", Telegram: value})
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
	for _, value := range []string{"javascript:alert(1)", "https://linkedin.com.evil/in/name", "https://evil@linkedin.com/in/name", "https://linkedin.com:443/in/name", "linkedin.com/in/a%2fb", "linkedin.com/in/name?x=1", "linkedin.com/in/name#x", "linkedin.com/company/name", "name", "recruiter/"} {
		t.Run("linkedin "+value, func(t *testing.T) {
			_, err := Normalize(Fields{Name: "N", LinkedIn: value})
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestContactWhitespaceAndUnicode(t *testing.T) {
	got, err := Normalize(Fields{Name: "\ufeffAda\ufeff"})
	require.NoError(t, err)
	require.Equal(t, "Ada", got.Name)
	_, err = Normalize(Fields{Name: "\u0085Ada"})
	require.ErrorIs(t, err, ErrInvalid)
	_, err = Normalize(Fields{Name: string([]byte{0xff})})
	require.ErrorIs(t, err, ErrInvalid)
	got, err = Normalize(Fields{Name: "\ufffd"})
	require.NoError(t, err)
	require.Equal(t, "\ufffd", got.Name)
}
