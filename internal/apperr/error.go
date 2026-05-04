package apperr

import (
	"errors"
	"fmt"
)

type reportedError struct {
	cause error
}

func Reported(cause error) error {
	if cause == nil {
		return nil
	}
	return reportedError{cause: cause}
}

func (e reportedError) Error() string {
	return e.cause.Error()
}

func (e reportedError) Unwrap() error {
	return e.cause
}

func IsReported(err error) bool {
	var reported reportedError
	return errors.As(err, &reported)
}

type Kind string

const (
	KindInvalidInput Kind = "invalid_input"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindForbidden    Kind = "forbidden"
)

type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(kind Kind, message string) error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, cause error) error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}

func InvalidInput(format string, args ...any) error {
	return New(KindInvalidInput, fmt.Sprintf(format, args...))
}

func NotFound(format string, args ...any) error {
	return New(KindNotFound, fmt.Sprintf(format, args...))
}

func Conflict(format string, args ...any) error {
	return New(KindConflict, fmt.Sprintf(format, args...))
}

func Forbidden(format string, args ...any) error {
	return New(KindForbidden, fmt.Sprintf(format, args...))
}
