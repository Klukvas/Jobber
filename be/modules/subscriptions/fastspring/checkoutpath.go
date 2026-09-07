package fastspring

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	// checkoutPathSegments is the exact shape of a dashboard checkout path.
	// The endpoint documents `checkoutPath` as "store-id/checkout-id", so any
	// other segment count is a configuration mistake rather than something to
	// forward to the provider.
	//
	// https://developer.fastspring.com/reference/createsession
	checkoutPathSegments = 2

	// PopupCheckoutPrefix is the prefix FastSpring gives a checkout generated as
	// a *popup* checkout. Jobber opens checkout through the Store Builder
	// Library popup, so a checkout id without this prefix names a full-page Web
	// Checkout and would leave the popup silently empty.
	//
	// https://developer.fastspring.com/reference/store-builder-library-overview
	PopupCheckoutPrefix = "popup-"

	// storefrontDomain is the domain every FastSpring-hosted storefront lives
	// on. It is the host family the Store Builder Library is pointed at.
	storefrontDomain = "onfastspring.com"

	// testStorefrontLabel is the extra label a *test* storefront carries:
	// "<store>.test.onfastspring.com" against live's "<store>.onfastspring.com".
	testStorefrontLabel = "test"
)

// checkoutPathSegment is the shape a single checkout-path segment may take.
// Anything else — a slash, a dot-segment, an encoded character, a query — is
// rejected rather than escaped, so a misconfigured value cannot be bent into a
// different API endpoint or a different storefront host.
var checkoutPathSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// storefrontLabel is the *stricter* shape the storefront id must take, because
// it becomes a DNS label in the storefront host. A label cannot contain a dot,
// so "fluxlab.example/popup-x" — a legal checkout path — must not be bent into
// "fluxlab.example.test.onfastspring.com".
var storefrontLabel = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)

// ErrInvalidCheckoutPath means the configured checkout path is not a plain
// segment list and must not be used to build a request URL.
var ErrInvalidCheckoutPath = errors.New("fastspring: invalid checkout path")

// ErrNotPopupCheckout means the configured checkout is not a popup checkout, so
// the Store Builder Library has nothing to open.
var ErrNotPopupCheckout = errors.New("fastspring: checkout path does not name a popup checkout")

// splitCheckoutPath validates a configured checkout path and returns its two
// parts. Validation is a whitelist rather than an escape, so no configured
// value can traverse to another endpoint or another host.
//
// The endpoint takes exactly "<storefront-id>/<checkout-id>". Surrounding
// slashes are not tolerated either: a leading slash would build
// "/v2/checkouts//store/..." and a trailing one leaves an empty final segment,
// so both are rejected rather than quietly repaired into a path the operator
// never configured.
func splitCheckoutPath(path string) (storefrontID, checkoutID string, err error) {
	segments := strings.Split(strings.TrimSpace(path), "/")
	if len(segments) != checkoutPathSegments {
		return "", "", fmt.Errorf(`%w: got %d segments, want exactly %d ("<storefront-id>/<checkout-id>")`,
			ErrInvalidCheckoutPath, len(segments), checkoutPathSegments)
	}
	for _, segment := range segments {
		if !checkoutPathSegment.MatchString(segment) {
			return "", "", fmt.Errorf("%w: segment %q is not a plain path segment", ErrInvalidCheckoutPath, segment)
		}
	}
	return segments[0], segments[1], nil
}

// EscapeCheckoutPath validates a configured checkout path segment by segment
// and returns it ready to be embedded in a request URL.
func EscapeCheckoutPath(path string) (string, error) {
	storefrontID, checkoutID, err := splitCheckoutPath(path)
	if err != nil {
		return "", err
	}
	return url.PathEscape(storefrontID) + "/" + url.PathEscape(checkoutID), nil
}

// ValidatePopupCheckoutPath reports whether a configured checkout path names a
// popup checkout. It is run at startup so a full-page Web Checkout path fails
// the boot instead of producing a popup that opens onto nothing.
func ValidatePopupCheckoutPath(path string) error {
	_, checkoutID, err := splitCheckoutPath(path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(checkoutID, PopupCheckoutPrefix) {
		return fmt.Errorf("%w: checkout id %q does not start with %q", ErrNotPopupCheckout, checkoutID, PopupCheckoutPrefix)
	}
	return nil
}

// PopupStorefront derives the Store Builder Library `data-storefront` value from
// the configured checkout path and the store mode.
//
//	fluxlab/popup-jobber, test → fluxlab.test.onfastspring.com/popup-jobber
//	fluxlab/popup-jobber, live → fluxlab.onfastspring.com/popup-jobber
//
// Deriving it keeps the storefront and the API checkout path from drifting apart
// in configuration, and means the browser is handed a host this code built from
// a validated value rather than a second free-text environment variable.
//
// https://developer.fastspring.com/reference/store-builder-library-overview
func PopupStorefront(checkoutPath string, live bool) (string, error) {
	storefrontID, checkoutID, err := splitCheckoutPath(checkoutPath)
	if err != nil {
		return "", err
	}
	if !storefrontLabel.MatchString(storefrontID) {
		return "", fmt.Errorf("%w: storefront id %q is not a single host label", ErrInvalidCheckoutPath, storefrontID)
	}
	if !strings.HasPrefix(checkoutID, PopupCheckoutPrefix) {
		return "", fmt.Errorf("%w: checkout id %q does not start with %q", ErrNotPopupCheckout, checkoutID, PopupCheckoutPrefix)
	}

	// Hostnames are case-insensitive, so the lowercase form is simply the
	// canonical one — it is not repairing an otherwise invalid value.
	labels := []string{strings.ToLower(storefrontID)}
	if !live {
		labels = append(labels, testStorefrontLabel)
	}
	labels = append(labels, storefrontDomain)

	return strings.Join(labels, ".") + "/" + checkoutID, nil
}
