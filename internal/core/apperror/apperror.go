// Package apperror defines transport-agnostic application errors.
//
// The core layer only describes *what* went wrong (Kind). The delivery layer
// decides how each Kind is represented (HTTP status, gRPC code, ...).
package apperror

import (
	"errors"
	"maps"
)

// Kind classifies an error independently of any transport.
type Kind uint8

const (
	KindInternal Kind = iota
	KindBadRequest
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindTooManyRequests
)

var defaultMessages = map[Kind]string{
	KindInternal:        "internal server error",
	KindBadRequest:      "bad request",
	KindValidation:      "validation failed",
	KindUnauthorized:    "unauthorized",
	KindForbidden:       "forbidden",
	KindNotFound:        "resource not found",
	KindConflict:        "resource conflict",
	KindTooManyRequests: "too many requests",
}

func (k Kind) String() string { return defaultMessages[k] }

// Error is the single error type used across the application.
type Error struct {
	Kind    Kind
	Message string
	// Fields holds per-field details (e.g. validation errors), keyed by field name.
	Fields map[string]string
	// Err is the underlying cause. It is never exposed to API clients.
	Err error
}

func (e *Error) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Kind.String()
	}
	if e.Err != nil {
		return msg + ": " + e.Err.Error()
	}
	return msg
}

// PublicMessage is the message that is safe to return to clients.
func (e *Error) PublicMessage() string {
	if e.Message == "" {
		return e.Kind.String()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// Is makes errors.Is match on Kind (and Message, when the target has one).
// This lets callers check either a broad category:
//
//	errors.Is(err, apperror.ErrNotFound)
//
// or a specific sentinel:
//
//	errors.Is(err, service.ErrInvalidCredentials)
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Kind == t.Kind && (t.Message == "" || e.Message == t.Message)
}

// WithFields returns a copy of the error with the given field details attached.
func (e *Error) WithFields(fields map[string]string) *Error {
	cp := *e
	cp.Fields = maps.Clone(fields)
	return &cp
}

// Wrap returns a copy of the error that carries cause as its underlying error.
func (e *Error) Wrap(cause error) *Error {
	cp := *e
	cp.Err = cause
	return &cp
}

// Category sentinels: match any error of the given kind via errors.Is.
var (
	ErrInternal        = &Error{Kind: KindInternal}
	ErrBadRequest      = &Error{Kind: KindBadRequest}
	ErrValidation      = &Error{Kind: KindValidation}
	ErrUnauthorized    = &Error{Kind: KindUnauthorized}
	ErrForbidden       = &Error{Kind: KindForbidden}
	ErrNotFound        = &Error{Kind: KindNotFound}
	ErrConflict        = &Error{Kind: KindConflict}
	ErrTooManyRequests = &Error{Kind: KindTooManyRequests}
)

func New(kind Kind, message string) *Error { return &Error{Kind: kind, Message: message} }

func Internal(cause error) *Error        { return &Error{Kind: KindInternal, Err: cause} }
func BadRequest(message string) *Error   { return New(KindBadRequest, message) }
func Unauthorized(message string) *Error { return New(KindUnauthorized, message) }
func Forbidden(message string) *Error    { return New(KindForbidden, message) }
func NotFound(message string) *Error     { return New(KindNotFound, message) }
func Conflict(message string) *Error     { return New(KindConflict, message) }
func TooManyRequests(msg string) *Error  { return New(KindTooManyRequests, msg) }
func Validation(f map[string]string) *Error {
	return &Error{Kind: KindValidation, Fields: maps.Clone(f)}
}

// As extracts an *Error from err. Unknown errors are reported as internal errors.
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Internal(err)
}
