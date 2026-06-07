// Package apperr provides application errors that combine a typed sentinel,
// optional context, and an optional wrapped cause.
//
// Sentinels must be comparable errors, usually package-local string types with
// const values. Error unwraps to both the sentinel and the wrapped cause, so
// callers can use errors.Is with either one. Callers that need the typed
// sentinel can use errors.As to recover *Error[S] and then call Sentinel.
package apperr

import "fmt"

type sentinel interface {
	error
	comparable
}

type Error[S sentinel] struct {
	sentinel S
	msg      string
	err      error
}

func (e *Error[S]) Error() string {
	if e == nil {
		return "<nil>"
	}

	base := e.sentinel.Error()
	if e.msg != "" {
		base += ": " + e.msg
	}

	if e.err != nil {
		base += ": " + e.err.Error()
	}

	return base
}

func (e *Error[S]) Unwrap() []error {
	if e == nil {
		return nil
	}

	errs := []error{e.sentinel}
	if e.err != nil {
		errs = append(errs, e.err)
	}

	return errs
}

func (e *Error[S]) Sentinel() S {
	if e == nil {
		var zero S
		return zero
	}

	return e.sentinel
}

func Wrap[S sentinel](sentinel S, err error, msg string) error {
	if err == nil && msg == "" {
		return sentinel
	}

	return &Error[S]{
		sentinel: sentinel,
		msg:      msg,
		err:      err,
	}
}

func Wrapf[S sentinel](sentinel S, err error, format string, args ...any) error {
	return Wrap(sentinel, err, fmt.Sprintf(format, args...))
}

func WrapErr[S sentinel](sentinel S, err error) error {
	return Wrap(sentinel, err, "")
}
