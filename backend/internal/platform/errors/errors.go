// Package errors defines the platform's domain-error vocabulary. Every
// service-layer function should return one of the *Error types declared
// here (or wrap a sentinel) so the HTTP layer can map them to status
// codes consistently.
//
// Design goals:
//   - explicit Kind for branching (no string sniffing)
//   - human-friendly Message safe to expose to API clients
//   - optional Cause preserved via errors.Is / errors.As
//   - field-level Details for validation feedback
package errors

import (
	stderrors "errors"
	"fmt"
)

// Kind enumerates broad categories of failure that map to HTTP status codes
// in interfaces/http.
type Kind string

const (
	KindUnknown      Kind = "unknown"
	KindValidation   Kind = "validation"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindRateLimited  Kind = "rate_limited"
	KindUnavailable  Kind = "unavailable"
	KindInternal     Kind = "internal"
)

// Error is the platform's structured error. Construct with the helpers below.
type Error struct {
	Kind    Kind                   `json:"kind"`
	Code    string                 `json:"code,omitempty"`
	Message string                 `json:"message"`
	Details map[string]string      `json:"details,omitempty"`
	Meta    map[string]interface{} `json:"-"`
	cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

// Unwrap supports errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.cause }

// WithCause attaches a low-level cause to the error.
func (e *Error) WithCause(cause error) *Error {
	if e == nil {
		return nil
	}
	e.cause = cause
	return e
}

// WithDetail attaches a single field-level detail (typical for validation).
func (e *Error) WithDetail(key, message string) *Error {
	if e == nil {
		return nil
	}
	if e.Details == nil {
		e.Details = make(map[string]string, 1)
	}
	e.Details[key] = message
	return e
}

// WithCode attaches a stable machine-readable code for clients to branch on.
func (e *Error) WithCode(code string) *Error {
	if e == nil {
		return nil
	}
	e.Code = code
	return e
}

// New constructs an Error of the given Kind.
func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

// Wrap turns any error into an Error of the given Kind, preserving the cause.
func Wrap(err error, kind Kind, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Message: message, cause: err}
}

// Convenience constructors -----------------------------------------------------

func Validation(message string) *Error   { return New(KindValidation, message) }
func Unauthorized(message string) *Error { return New(KindUnauthorized, message) }
func Forbidden(message string) *Error    { return New(KindForbidden, message) }
func NotFound(message string) *Error     { return New(KindNotFound, message) }
func Conflict(message string) *Error     { return New(KindConflict, message) }
func RateLimited(message string) *Error  { return New(KindRateLimited, message) }
func Unavailable(message string) *Error  { return New(KindUnavailable, message) }
func Internal(message string) *Error     { return New(KindInternal, message) }

// As is a convenience over errors.As for the common case of inspecting an *Error.
func As(err error) (*Error, bool) {
	var e *Error
	if stderrors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// KindOf returns the Kind of the underlying *Error, or KindUnknown.
func KindOf(err error) Kind {
	if e, ok := As(err); ok {
		return e.Kind
	}
	return KindUnknown
}
