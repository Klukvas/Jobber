// Package ids normalises caller-supplied identifiers before they reach storage.
package ids

import "github.com/google/uuid"

// CanonicalUUID reports whether raw is a UUID and returns it in canonical form:
// lowercase, hyphenated, 36 characters.
//
// uuid.Parse is more lenient than anything downstream of it. It also accepts
// `urn:uuid:<uuid>`, `{<uuid>}` and the 32-digit compact form, and PostgreSQL's
// uuid input rejects the URN form outright, so a request that passed validation
// would still fail as a 500. The other forms reach Postgres fine but differ from
// the stored value as text: in storage keys, idempotency receipts and Go-side
// string comparisons. Forward the returned value, never the original string.
func CanonicalUUID(raw string) (string, bool) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}

// CanonicalUUIDs is CanonicalUUID for several ids at once: it returns them in
// canonical form and in the same order, or false when any of them is not a UUID.
func CanonicalUUIDs(raw ...string) ([]string, bool) {
	canonical := make([]string, len(raw))
	for i, id := range raw {
		value, ok := CanonicalUUID(id)
		if !ok {
			return nil, false
		}
		canonical[i] = value
	}
	return canonical, true
}
