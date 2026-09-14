package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentDisposition(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain ascii title",
			in:   "Backend Engineer",
			want: `attachment; filename="Backend Engineer.pdf"; filename*=UTF-8''Backend%20Engineer.pdf`,
		},
		{
			name: "an existing .pdf suffix is not doubled",
			in:   "Backend Engineer.pdf",
			want: `attachment; filename="Backend Engineer.pdf"; filename*=UTF-8''Backend%20Engineer.pdf`,
		},
		{
			// Titles arrive as "CV.PDF" as often as "cv.pdf"; a case-sensitive
			// trim left the header reading "CV.PDF.pdf".
			name: "the suffix is stripped whatever its case",
			in:   "Backend Engineer.PDF",
			want: `attachment; filename="Backend Engineer.pdf"; filename*=UTF-8''Backend%20Engineer.pdf`,
		},
		{
			name: "a mixed-case suffix too",
			in:   "Backend Engineer.Pdf",
			want: `attachment; filename="Backend Engineer.pdf"; filename*=UTF-8''Backend%20Engineer.pdf`,
		},
		{
			// ".pdf" as the whole title leaves nothing behind, which falls back
			// the same way an empty title does.
			name: "a title that is only the suffix",
			in:   ".PDF",
			want: "",
		},
		{
			name: "non-ascii survives in filename* and is transliterated in the fallback",
			in:   "Резюме",
			want: `attachment; filename="______.pdf"; filename*=UTF-8''%D0%A0%D0%B5%D0%B7%D1%8E%D0%BC%D0%B5.pdf`,
		},
		{
			// The apostrophe is the ext-value's own delimiter, so leaving it
			// raw (as url.PathEscape did) made the receiver split the value in
			// the wrong place and read a bogus charset.
			name: "an apostrophe is percent-encoded, not left as a delimiter",
			in:   "John's CV",
			want: `attachment; filename="John's CV.pdf"; filename*=UTF-8''John%27s%20CV.pdf`,
		},
		{
			name: "the other characters url.PathEscape leaves raw are encoded too",
			in:   "a(b)c*d,e:f@g=h&i+j$k",
			want: `attachment; filename="a(b)c*d,e:f@g=h&i+j$k.pdf"; filename*=UTF-8''a%28b%29c%2Ad%2Ce%3Af%40g%3Dh%26i%2Bj%24k.pdf`,
		},
		{
			name: "attr-chars that are safe stay readable",
			in:   "cv-2026_final.v2!~",
			want: `attachment; filename="cv-2026_final.v2!~.pdf"; filename*=UTF-8''cv-2026_final.v2!~.pdf`,
		},
		{
			name: "a percent sign is escaped so the value cannot be mis-decoded",
			in:   "100% match",
			want: `attachment; filename="100% match.pdf"; filename*=UTF-8''100%25%20match.pdf`,
		},
		{
			name: "mixed scripts encode byte by byte",
			in:   "CV Резюме 履歴書",
			want: `attachment; filename="CV ______ ___.pdf"; filename*=UTF-8''CV%20%D0%A0%D0%B5%D0%B7%D1%8E%D0%BC%D0%B5%20%E5%B1%A5%E6%AD%B4%E6%9B%B8.pdf`,
		},
		{
			name: "empty title means no override",
			in:   "",
			want: "",
		},
		{
			name: "a title that is only punctuation means no override",
			in:   " ... ",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AttachmentDisposition(tt.in))
		})
	}
}

// A resume title is free text, so it must never be able to close the quoted
// filename parameter or inject a path.
func TestAttachmentDispositionNeutralisesHeaderAndPathSyntax(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "double quote", in: `evil".pdf`},
		{name: "semicolon", in: "evil; attachment"},
		{name: "posix path", in: "../../etc/passwd"},
		{name: "windows path", in: `..\..\secret`},
		{name: "crlf injection", in: "evil\r\nX-Header: 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AttachmentDisposition(tt.in)
			require.NotEmpty(t, got)

			quoted := strings.TrimPrefix(got, `attachment; filename="`)
			quoted = quoted[:strings.Index(quoted, `"`)]

			assert.NotContains(t, quoted, `"`)
			assert.NotContains(t, quoted, ";")
			assert.NotContains(t, quoted, "/")
			assert.NotContains(t, quoted, `\`)
			assert.NotContains(t, got, "\r")
			assert.NotContains(t, got, "\n")
		})
	}
}

func TestAttachmentDispositionTruncatesVeryLongTitles(t *testing.T) {
	got := AttachmentDisposition(strings.Repeat("a", 500))

	require.NotEmpty(t, got)
	assert.Contains(t, got, strings.Repeat("a", maxDownloadNameLen)+".pdf")
	assert.NotContains(t, got, strings.Repeat("a", maxDownloadNameLen+1))
}

// Slicing by rune can land between a letter and its combining accent, leaving
// the truncated name ending in an orphan mark that renders on whatever
// character the client puts before it.
func TestAttachmentDispositionDoesNotLeaveAnOrphanCombiningMark(t *testing.T) {
	// 99 plain letters, then "e" + U+0301 COMBINING ACUTE: the cut falls
	// exactly between the two.
	title := strings.Repeat("a", maxDownloadNameLen-1) + "e\u0301xtra"

	got := AttachmentDisposition(title)

	require.NotEmpty(t, got)
	assert.NotContains(t, got, "\u0301")
	assert.Contains(t, got, strings.Repeat("a", maxDownloadNameLen-1)+".pdf")
}

func TestGetObjectPrefix(t *testing.T) {
	body := append([]byte("%PDF-1.7\n"), make([]byte, 4096)...)
	objects := map[string][]byte{
		"users/u1/resumes/r1.pdf": body,
		"users/u1/resumes/tiny":   []byte("%PDF"),
		// Present in the bucket, zero bytes long — a PUT that uploaded nothing.
		"users/u1/resumes/empty": {},
	}
	client, cleanup := NewTestS3Client(objects)
	defer cleanup()

	t.Run("returns only the prefix but the full size", func(t *testing.T) {
		data, size, err := client.GetObjectPrefix(context.Background(), "users/u1/resumes/r1.pdf", 16)

		require.NoError(t, err)
		assert.Len(t, data, 16)
		assert.Equal(t, "%PDF-1.7", string(data[:8]))
		assert.Equal(t, int64(len(body)), size)
	})

	t.Run("handles an object smaller than the requested prefix", func(t *testing.T) {
		data, size, err := client.GetObjectPrefix(context.Background(), "users/u1/resumes/tiny", 1024)

		require.NoError(t, err)
		assert.Equal(t, "%PDF", string(data))
		assert.Equal(t, int64(4), size)
	})

	// A zero-byte object cannot satisfy any range, so real S3 and Hetzner answer
	// 416 InvalidRange rather than 404. That has to read as "an object with no
	// bytes", not as a failure — otherwise the size check downstream never runs
	// and the customer gets a 500 for an empty upload.
	t.Run("reports a present but empty object as a successful read of nothing", func(t *testing.T) {
		data, size, err := client.GetObjectPrefix(context.Background(), "users/u1/resumes/empty", 1024)

		require.NoError(t, err)
		assert.Empty(t, data)
		assert.Equal(t, int64(0), size)
	})

	t.Run("an empty object is not confused with a missing one", func(t *testing.T) {
		_, _, emptyErr := client.GetObjectPrefix(context.Background(), "users/u1/resumes/empty", 1024)
		_, _, missingErr := client.GetObjectPrefix(context.Background(), "users/u1/resumes/gone", 1024)

		assert.NoError(t, emptyErr)
		assert.ErrorIs(t, missingErr, ErrObjectNotFound)
	})

	// Callers have to be able to tell "no such file" from "the store is
	// unreachable": the first is the customer's problem (a 400), the second is
	// ours (a 500).
	t.Run("reports a missing object as ErrObjectNotFound", func(t *testing.T) {
		_, _, err := client.GetObjectPrefix(context.Background(), "users/u1/resumes/gone", 1024)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrObjectNotFound)
		assert.Contains(t, err.Error(), "users/u1/resumes/gone")
	})

	t.Run("does not label a transport failure as not-found", func(t *testing.T) {
		downClient, downCleanup := NewTestS3Client(map[string][]byte{})
		downCleanup() // shut the server down before the read

		_, _, err := downClient.GetObjectPrefix(context.Background(), "users/u1/resumes/r1.pdf", 1024)

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrObjectNotFound)
	})

	t.Run("does not label a cancelled read as not-found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := client.GetObjectPrefix(ctx, "users/u1/resumes/r1.pdf", 1024)

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrObjectNotFound)
	})

	t.Run("rejects a non-positive length", func(t *testing.T) {
		_, _, err := client.GetObjectPrefix(context.Background(), "users/u1/resumes/r1.pdf", 0)

		assert.Error(t, err)
	})
}

// The prefix read is a *size cap* input: whatever comes back is compared with
// MaxUploadedResumeBytes and rejected when it is over. So a response that does
// not state the object's real size must never be answered with "however much
// we happened to read" — that reported a 20 MB object as a 1 KB one and waved
// it straight through the cap.
func TestGetObjectPrefixSizeWhenTheResponseDoesNotStateIt(t *testing.T) {
	const oversized = 20 * 1024 * 1024
	big := append([]byte("%PDF-1.7\n"), make([]byte, oversized)...)
	const key = "users/u1/resumes/big.pdf"

	newClient := func(t *testing.T, opts TestS3Options) *S3Client {
		t.Helper()
		client, cleanup := NewTestS3ClientWithOptions(
			map[string][]byte{key: big},
			opts,
		)
		t.Cleanup(cleanup)
		return client
	}

	t.Run("falls back to HeadObject when the total is a star", func(t *testing.T) {
		client := newClient(t, TestS3Options{
			ContentRangeOverride: map[string]string{key: "bytes 0-1023/*"},
		})

		_, size, err := client.GetObjectPrefix(context.Background(), key, 1024)

		require.NoError(t, err)
		assert.Equal(t, int64(len(big)), size)
	})

	t.Run("falls back to HeadObject when the total does not parse", func(t *testing.T) {
		for name, header := range map[string]string{
			"not a number":  "bytes 0-1023/not-a-number",
			"no slash":      "bytes 0-1023",
			"negative":      "bytes 0-1023/-5",
			"empty total":   "bytes 0-1023/",
			"header absent": "",
		} {
			t.Run(name, func(t *testing.T) {
				client := newClient(t, TestS3Options{
					ContentRangeOverride: map[string]string{key: header},
				})

				_, size, err := client.GetObjectPrefix(context.Background(), key, 1024)

				require.NoError(t, err)
				assert.Equal(t, int64(len(big)), size)
			})
		}
	})

	// A store that ignores the Range and streams the object chunked states no
	// length anywhere, and the read stops at the limit — indistinguishable from
	// a 1 KB file without asking.
	t.Run("falls back to HeadObject when a whole-object response has no length", func(t *testing.T) {
		client := newClient(t, TestS3Options{
			IgnoreRange:       map[string]bool{key: true},
			OmitContentLength: map[string]bool{key: true},
		})

		data, size, err := client.GetObjectPrefix(context.Background(), key, 1024)

		require.NoError(t, err)
		assert.Len(t, data, 1024)
		assert.Equal(t, int64(len(big)), size)
	})

	// Fail closed: an unknown size must not become a small one.
	t.Run("returns an error when the size cannot be established", func(t *testing.T) {
		client := newClient(t, TestS3Options{
			ContentRangeOverride: map[string]string{key: "bytes 0-1023/*"},
			FailHead:             map[string]bool{key: true},
		})

		_, _, err := client.GetObjectPrefix(context.Background(), key, 1024)

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrObjectNotFound)
		assert.Contains(t, err.Error(), key)
	})

	// The paths that already stated a size must keep answering from the
	// response, with no extra round-trip and no behaviour change.
	t.Run("still reads the total straight from a well-formed Content-Range", func(t *testing.T) {
		client := newClient(t, TestS3Options{
			// A HEAD would fail, so passing proves none was made.
			FailHead: map[string]bool{key: true},
		})

		_, size, err := client.GetObjectPrefix(context.Background(), key, 1024)

		require.NoError(t, err)
		assert.Equal(t, int64(len(big)), size)
	})

	t.Run("still reads the total from Content-Length on a whole-object response", func(t *testing.T) {
		client := newClient(t, TestS3Options{
			IgnoreRange: map[string]bool{key: true},
			FailHead:    map[string]bool{key: true},
		})

		_, size, err := client.GetObjectPrefix(context.Background(), key, 1024)

		require.NoError(t, err)
		assert.Equal(t, int64(len(big)), size)
	})

	// A short object read to its end needs no help: the body ended before the
	// limit, so its length is the whole truth.
	t.Run("does not head an object that ended before the limit", func(t *testing.T) {
		const shortKey = "users/u1/resumes/short.pdf"
		short := []byte("%PDF-1.7\n")
		client, cleanup := NewTestS3ClientWithOptions(
			map[string][]byte{shortKey: short},
			TestS3Options{
				IgnoreRange:       map[string]bool{shortKey: true},
				OmitContentLength: map[string]bool{shortKey: true},
				FailHead:          map[string]bool{shortKey: true},
			},
		)
		defer cleanup()

		data, size, err := client.GetObjectPrefix(context.Background(), shortKey, 1024)

		require.NoError(t, err)
		assert.Equal(t, short, data)
		assert.Equal(t, int64(len(short)), size)
	})

	// A zero-byte object never reaches the size resolution at all — the 416 is
	// answered before it — so this stays a successful read of nothing.
	t.Run("leaves the zero-byte object path untouched", func(t *testing.T) {
		const emptyKey = "users/u1/resumes/empty"
		client, cleanup := NewTestS3ClientWithOptions(
			map[string][]byte{emptyKey: {}},
			TestS3Options{FailHead: map[string]bool{emptyKey: true}},
		)
		defer cleanup()

		data, size, err := client.GetObjectPrefix(context.Background(), emptyKey, 1024)

		require.NoError(t, err)
		assert.Empty(t, data)
		assert.Equal(t, int64(0), size)
	})
}

func TestTotalFromContentRange(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		want   int64
		wantOK bool
	}{
		{name: "well formed", in: "bytes 0-1023/54321", want: 54321, wantOK: true},
		{name: "whole object", in: "bytes 0-3/4", want: 4, wantOK: true},
		{name: "surrounding space", in: "bytes 0-1023/ 54321 ", want: 54321, wantOK: true},
		{name: "unknown total", in: "bytes 0-1023/*"},
		{name: "no slash", in: "bytes 0-1023"},
		{name: "not a number", in: "bytes 0-1023/abc"},
		{name: "negative", in: "bytes 0-1023/-1"},
		{name: "empty", in: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := totalFromContentRange(tt.in)

			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
