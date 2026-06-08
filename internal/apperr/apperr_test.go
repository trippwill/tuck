package apperr

import (
	"errors"
	"testing"
)

type testErr string

const (
	errInvalid testErr = "invalid"
	errMissing testErr = "missing"
)

func (e testErr) Error() string {
	return string(e)
}

func TestAppErrMsgReturnsSentinelWithoutContext(t *testing.T) {
	err := AppErrMsg(errInvalid, "")

	if err != errInvalid {
		t.Fatalf("AppErrMsg() = %#v, want sentinel", err)
	}
}

func TestAppErrWrapfSupportsErrorsIsForSentinelAndCause(t *testing.T) {
	cause := errors.New("cause")
	err := AppErrWrapf(errInvalid, cause, "with %s", "context")

	if !errors.Is(err, errInvalid) {
		t.Fatalf("errors.Is(err, errInvalid) = false, want true")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(err, cause) = false, want true")
	}
	if errors.Is(err, errMissing) {
		t.Fatalf("errors.Is(err, errMissing) = true, want false")
	}
}

func TestAppErrMsgfSupportsErrorsAsAndSentinel(t *testing.T) {
	err := AppErrMsgf(errInvalid, "bad %s", "input")

	var appErr *Error[testErr]
	if !errors.As(err, &appErr) {
		t.Fatalf("errors.As(err, *Error[testErr]) = false, want true")
	}
	if got := appErr.Sentinel(); got != errInvalid {
		t.Fatalf("Sentinel() = %v, want %v", got, errInvalid)
	}
}

func TestAppErrMsgfSupportsErrorsAsTypeAndSentinel(t *testing.T) {
	err := AppErrMsgf(errInvalid, "bad %s", "input")

	appErr, ok := errors.AsType[*Error[testErr]](err)
	if !ok {
		t.Fatalf("errors.AsType[*Error[testErr]](err) ok = false, want true")
	}
	if got := appErr.Sentinel(); got != errInvalid {
		t.Fatalf("Sentinel() = %v, want %v", got, errInvalid)
	}
}

func TestAppErrWrapUsesCauseWithoutMessage(t *testing.T) {
	cause := errors.New("disk failed")
	err := AppErrWrap(errMissing, cause)

	if got, want := err.Error(), "missing: disk failed"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorStringIncludesSentinelMessageAndCause(t *testing.T) {
	err := AppErrWrapf(errInvalid, errors.New("toml failed"), "could not %s", "parse")

	if got, want := err.Error(), "invalid: could not parse: toml failed"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestNilErrorReceiver(t *testing.T) {
	var err *Error[testErr]

	if got, want := err.Error(), "<nil>"; got != want {
		t.Fatalf("nil Error() = %q, want %q", got, want)
	}
	if got := err.Unwrap(); got != nil {
		t.Fatalf("nil Unwrap() = %#v, want nil", got)
	}
	if got := err.Sentinel(); got != "" {
		t.Fatalf("nil Sentinel() = %q, want zero value", got)
	}
}
