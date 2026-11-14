// Package idstest holds UUID fixtures for regression tests of id handling.
package idstest

import "strings"

// NonCanonicalForms returns the spellings of canonical that uuid.Parse accepts
// besides canonical itself, keyed by a short name for subtests. canonical must
// be a lowercase, hyphenated UUID.
func NonCanonicalForms(canonical string) map[string]string {
	return map[string]string{
		"urn":       "urn:uuid:" + canonical,
		"braces":    "{" + canonical + "}",
		"uppercase": strings.ToUpper(canonical),
		"compact":   strings.ReplaceAll(canonical, "-", ""),
	}
}
