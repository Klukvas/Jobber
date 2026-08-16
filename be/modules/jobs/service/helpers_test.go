package service

import "time"

// timeNowUTC is a small helper for building deterministic *time.Time values in
// tests. The service stamps its own timestamps internally, so tests only need a
// concrete instant to hand to the mocks.
func timeNowUTC() time.Time {
	return time.Now().UTC()
}

// stringPtrMatcher matches a *string SQL argument by its dereferenced value.
// want == nil means "match a nil *string". It implements pgxmock's Argument
// interface so tests can assert on pointer args the service passes to Exec.
type stringPtrMatcher struct {
	want *string
}

func (m stringPtrMatcher) Match(v interface{}) bool {
	p, ok := v.(*string)
	if !ok {
		return false
	}
	if m.want == nil {
		return p == nil
	}
	return p != nil && *p == *m.want
}

// strPtrArg matches a non-nil *string arg equal to want.
func strPtrArg(want string) stringPtrMatcher {
	return stringPtrMatcher{want: &want}
}

// nilStrPtrArg matches a nil *string arg.
func nilStrPtrArg() stringPtrMatcher {
	return stringPtrMatcher{want: nil}
}

// nilTimePtrMatcher matches a nil *time.Time SQL argument. It implements
// pgxmock's Argument interface, letting tests assert applied_at was left unset.
type nilTimePtrMatcher struct{}

func (nilTimePtrMatcher) Match(v interface{}) bool {
	p, ok := v.(*time.Time)
	return ok && p == nil
}

func nilTimePtrArg() nilTimePtrMatcher { return nilTimePtrMatcher{} }
