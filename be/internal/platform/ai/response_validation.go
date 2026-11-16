package ai

import "fmt"

// ResponseValidationError describes a rejected AI response without retaining any
// source text, prompt, response or provider error body. Cause is a domain sentinel.
type ResponseValidationError struct {
	Stage string
	Rule  string
	Path  string
	Cause error
}

func (e *ResponseValidationError) Error() string {
	return fmt.Sprintf("invalid AI response: stage=%s rule=%s path=%s", e.Stage, e.Rule, e.Path)
}

func (e *ResponseValidationError) Unwrap() error { return e.Cause }

// InvalidResponse preserves sentinel matching while exposing safe diagnostics.
func InvalidResponse(cause error, stage, rule, path string) error {
	return &ResponseValidationError{Stage: stage, Rule: rule, Path: path, Cause: cause}
}
