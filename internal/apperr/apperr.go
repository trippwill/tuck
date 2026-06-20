// Package apperr provides application errors that combine a typed sentinel,
// optional context, and an optional wrapped cause.
//
// Sentinels must be comparable errors, usually package-local string types with
// const values. Error unwraps to both the sentinel and the wrapped cause, so
// callers normally classify failures with errors.Is and the package sentinel.
//
// Callers that need typed app-error metadata can name *apperr.Error[pkg.ErrKind]
// directly.
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

func AppErrMsg[S sentinel](sentinel S, msg string) error {
	return appErr(sentinel, nil, msg)
}

func AppErrMsgf[S sentinel](sentinel S, format string, args ...any) error {
	return AppErrMsg(sentinel, fmt.Sprintf(format, args...))
}

func AppErrWrap[S sentinel](sentinel S, err error) error {
	return appErr(sentinel, err, "")
}

func AppErrWrapf[S sentinel](sentinel S, err error, format string, args ...any) error {
	return appErr(sentinel, err, fmt.Sprintf(format, args...))
}

func appErr[S sentinel](sentinel S, cause error, msg string) error {
	if cause == nil && msg == "" {
		return sentinel
	}

	return &Error[S]{
		sentinel: sentinel,
		msg:      msg,
		err:      cause,
	}
}
