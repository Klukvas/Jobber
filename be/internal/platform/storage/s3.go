package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Client provides S3 storage operations
type S3Client struct {
	client *s3.Client
	bucket string
}

// NewS3Client creates a new S3 client with custom endpoint support
func NewS3Client(cfg config.S3Config) (*S3Client, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("S3 configuration is incomplete")
	}

	// Create custom resolver for Hetzner endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// Create AWS config with custom credentials and endpoint
	awsConfig := aws.Config{
		Region:                      cfg.Region,
		Credentials:                 credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		EndpointResolverWithOptions: customResolver,
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true // Required for S3-compatible storage
	})

	return &S3Client{
		client: s3Client,
		bucket: cfg.Bucket,
	}, nil
}

// GeneratePresignedUploadURL generates a presigned URL for uploading a file.
//
// NOTE on signed headers: aws-sdk-go-v2's presigner signs ONLY the host header
// for presigned PUT URLs (the resulting query has X-Amz-SignedHeaders=host).
// The contentType passed here is NOT added to the signature, so the client may
// send any Content-Type (or none) without triggering a signature mismatch —
// whatever Content-Type the client sends on the actual PUT is what the object
// is stored with. We still pass it to document the expected type.
func (c *S3Client) GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)

	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return request.URL, nil
}

// GeneratePresignedDownloadURL generates a presigned URL for downloading a
// file. When downloadName is non-empty the URL also carries a
// Content-Disposition override, so the browser saves the file under that name
// instead of the opaque storage key.
func (c *S3Client) GeneratePresignedDownloadURL(ctx context.Context, key string, downloadName string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	if disposition := AttachmentDisposition(downloadName); disposition != "" {
		input.ResponseContentDisposition = aws.String(disposition)
	}

	request, err := presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return request.URL, nil
}

// AttachmentDisposition builds an RFC 6266 Content-Disposition value for a
// user-supplied file name.
//
// The name reaches this function straight from a resume title, so it is
// stripped of path separators, quotes and control characters before it is
// embedded in the quoted ASCII fallback; the exact name travels in the
// percent-encoded `filename*` parameter, which is what modern browsers use.
// An empty result means "no override" — the caller keeps the storage default.
func AttachmentDisposition(name string) string {
	sanitized := sanitizeDownloadName(name)
	if sanitized == "" {
		return ""
	}

	ascii := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return '_'
		}
		return r
	}, sanitized)

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s",
		ascii, encodeExtValue(sanitized))
}

// extValueSafe is the set of bytes that may appear unencoded in an RFC 8187
// ext-value. It is a deliberately strict subset of the RFC's attr-char:
// over-encoding is always safe to decode, under-encoding is not.
//
// `url.PathEscape` is not usable here — it leaves "'", "(", ")", "*", ",",
// ":", "@", "=", "&", "+", "$" alone, and the apostrophe in particular is the
// parameter's own delimiter, so a title like "John's CV" produced a value the
// receiver split in the wrong place.
func extValueSafe(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '!', b == '#', b == '-', b == '.', b == '^', b == '_', b == '`', b == '|', b == '~':
		return true
	default:
		return false
	}
}

const upperHex = "0123456789ABCDEF"

// encodeExtValue percent-encodes a UTF-8 string for the value part of an
// RFC 8187 ext-value: the part that follows the charset and language
// delimiters. Encoding is per BYTE, so multi-byte characters become one
// %XX group per byte.
func encodeExtValue(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(s); i++ {
		b := s[i]
		if extValueSafe(b) {
			out.WriteByte(b)
			continue
		}
		out.WriteByte('%')
		out.WriteByte(upperHex[b>>4])
		out.WriteByte(upperHex[b&0x0F])
	}
	return out.String()
}

// maxDownloadNameLen keeps the generated header well inside what browsers and
// proxies accept once the name is percent-encoded.
const maxDownloadNameLen = 100

// pdfExtension is the suffix every generated download name ends with, and the
// one an incoming title is stripped of first so it is never doubled.
const pdfExtension = ".pdf"

// sanitizeDownloadName reduces a free-text title to something safe to use as a
// file name, and gives it a .pdf extension.
func sanitizeDownloadName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r < 32 || r == 127: // control characters
			return -1
		case r == '/' || r == '\\' || r == '"' || r == ';': // path + header syntax
			return '-'
		default:
			return r
		}
	}, name)

	cleaned = strings.TrimSpace(cleaned)
	// Case-insensitive: titles arrive as "CV.PDF" as often as "cv.pdf", and a
	// case-sensitive trim left the header reading "CV.PDF.pdf".
	if len(cleaned) >= len(pdfExtension) &&
		strings.EqualFold(cleaned[len(cleaned)-len(pdfExtension):], pdfExtension) {
		cleaned = cleaned[:len(cleaned)-len(pdfExtension)]
	}
	cleaned = strings.TrimSpace(strings.Trim(cleaned, "."))
	if cleaned == "" {
		return ""
	}

	if runes := []rune(cleaned); len(runes) > maxDownloadNameLen {
		// Truncated rather than split across RFC 2231 `filename*0*`/`*1*`
		// continuations: the continuation form is poorly supported by real
		// download managers, and a name this long is decoration on a file the
		// customer already named.
		//
		// The cut is by rune, which can land inside a combining sequence and
		// leave a letter stripped of the accent that belongs to it — "é" ending
		// up as "e", a different letter. When the first dropped rune is a
		// combining mark the whole cluster goes instead of half of it.
		cut := maxDownloadNameLen
		for cut > 0 && unicode.Is(unicode.M, runes[cut]) {
			cut--
		}
		cleaned = strings.TrimSpace(string(runes[:cut]))
		if cleaned == "" {
			return ""
		}
	}

	return cleaned + pdfExtension
}

// DeleteObject deletes an object from S3
func (c *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GetObject downloads an object from S3 and returns its contents
func (c *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}

// ErrObjectNotFound reports that a key does not exist in the bucket, as
// distinct from any other reason a read can fail. Callers need the difference:
// a missing object is a customer-visible "there is no file here", while a
// timeout or a permission problem is an internal fault and must not be
// presented as one.
var ErrObjectNotFound = errors.New("object not found")

// isNotFound reports whether an SDK error means "this key does not exist".
//
// Both shapes are checked on purpose: AWS models NoSuchKey as a typed error,
// but S3-compatible stores (Hetzner here) sometimes answer a ranged GET with a
// bare HTTP 404 that never decodes into that type.
func isNotFound(err error) bool {
	var noKey *s3types.NoSuchKey
	if errors.As(err, &noKey) {
		return true
	}
	var notFound *s3types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var responseErr *awshttp.ResponseError
	if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	return false
}

// apiError is the shape smithy's APIError satisfies. Declared locally so this
// package can read the service error code without taking a direct dependency
// on smithy-go, which the module only carries indirectly.
type apiError interface {
	ErrorCode() string
}

// isUnsatisfiableRange reports whether a ranged read failed because the range
// covers nothing.
//
// Only meaningful for a range that STARTS AT ZERO, which is the only kind this
// file issues. Per RFC 9110 a byte range with first-pos 0 is satisfiable
// whenever the representation has any bytes at all, so "unsatisfiable" here can
// only mean the object is zero length — it can never mean an out-of-bounds read
// of a non-empty object.
//
// Both signals are accepted because stores disagree on the shape: AWS answers
// with the InvalidRange service code, while some S3-compatible stores return a
// bare HTTP 416.
func isUnsatisfiableRange(err error) bool {
	var apiErr apiError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidRange" {
		return true
	}
	var responseErr *awshttp.ResponseError
	if errors.As(err, &responseErr) &&
		responseErr.HTTPStatusCode() == http.StatusRequestedRangeNotSatisfiable {
		return true
	}
	return false
}

// GetObjectPrefix reads at most maxBytes from the start of an object and
// reports the object's full size. It exists so a content check (magic bytes,
// size) can run against a freshly uploaded object without pulling the whole
// file back from storage.
//
// A zero-length object is reported as (nil, 0, nil) rather than as an error:
// the range always starts at byte 0, so a store rejecting it as unsatisfiable
// is telling us the object has no bytes. That is a fact about the object, not a
// failure to read it, and callers validate emptiness themselves.
//
// A missing key is reported as ErrObjectNotFound; every other failure is
// wrapped as-is so the caller can treat it as the internal error it is.
//
// The size comes from the response's Content-Range (`bytes 0-1023/54321`) when
// the store honours the Range header, and from Content-Length otherwise — some
// S3-compatible stores serve the whole object and ignore the range. When
// neither states it (a `*` total, a malformed header, or a body that filled
// the limit with no length at all) the size is fetched with a HeadObject, and
// a failure there is returned: callers use this number as a size cap, so
// guessing low would let an oversized object through.
func (c *S3Client) GetObjectPrefix(ctx context.Context, key string, maxBytes int64) ([]byte, int64, error) {
	if maxBytes <= 0 {
		return nil, 0, fmt.Errorf("maxBytes must be positive, got %d", maxBytes)
	}

	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Range:  aws.String(fmt.Sprintf("bytes=0-%d", maxBytes-1)),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, 0, fmt.Errorf("get object prefix %q: %w", key, ErrObjectNotFound)
		}
		// The object exists but has no bytes — the only way a range starting at
		// zero can be unsatisfiable. Reported as a successful read of nothing.
		if isUnsatisfiableRange(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to get object prefix: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(io.LimitReader(output.Body, maxBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read object prefix: %w", err)
	}

	totalSize, err := c.resolveTotalSize(ctx, key, output.ContentRange, output.ContentLength, int64(len(data)), maxBytes)
	if err != nil {
		return nil, 0, err
	}

	return data, totalSize, nil
}

// totalFromContentRange reads the total after the "/" of a Content-Range value.
// Returns false for the unsatisfied form ("bytes */1234" carries a total, but
// "bytes 0-1023/*" does not) and for anything that does not parse.
func totalFromContentRange(contentRange string) (int64, bool) {
	_, after, found := strings.Cut(contentRange, "/")
	if !found {
		return 0, false
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(after), 10, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}

// resolveTotalSize works out how big the object really is, and refuses to
// guess.
//
// This is a size *limit* input, not a statistic: callers compare the result
// against a maximum and reject what is over it. Falling back to "however many
// bytes we happened to read" — which is what an unparseable or `*` total used
// to produce — reports a 50 MB object as a 1 KB one and waves it straight
// through the cap. So when the response does not state the size, the size is
// asked for with a HeadObject, and a failure there is an error rather than a
// permissive default.
func (c *S3Client) resolveTotalSize(
	ctx context.Context,
	key string,
	contentRange *string,
	contentLength *int64,
	read int64,
	maxBytes int64,
) (int64, error) {
	if contentRange != nil {
		if total, ok := totalFromContentRange(*contentRange); ok {
			return total, nil
		}
		// A range response whose total is "*" or malformed: the body is a
		// prefix of unknown length. Ask outright.
		return c.headContentLength(ctx, key)
	}

	// With no Content-Range, Content-Length is the only clue — and it describes
	// whatever body was actually sent, which is the whole object when the store
	// ignored the range and just the chunk when it did not. Only a value larger
	// than what was read can be read as "the object is bigger than this
	// request", so that is the only case it is trusted for.
	if contentLength != nil && *contentLength > read {
		return *contentLength, nil
	}

	if read < maxBytes {
		// The body ended before the limit, so the object was read to its end
		// and its length is exactly what came back.
		return read, nil
	}

	// The read stopped exactly at the limit with nothing distinguishing "an
	// object of precisely this size" from "the first slice of a much bigger
	// one". Ask instead of assuming the smaller of the two.
	return c.headContentLength(ctx, key)
}

// headContentLength asks the store for an object's size. Used only when a read
// response did not state it; a failure is returned rather than swallowed so a
// size check never runs on an invented number.
func (c *S3Client) headContentLength(ctx context.Context, key string) (int64, error) {
	head, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return 0, fmt.Errorf("get object prefix %q: %w", key, ErrObjectNotFound)
		}
		return 0, fmt.Errorf("failed to determine size of object %q: %w", key, err)
	}
	if head.ContentLength == nil {
		return 0, fmt.Errorf("storage did not report a size for object %q", key)
	}
	return *head.ContentLength, nil
}

// ObjectExists checks if an object exists in S3
func (c *S3Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nf *s3types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		// A transient/permission error must NOT be reported as "not found".
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}
