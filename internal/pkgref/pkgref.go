package pkgref

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/trippwill/tuck/internal/apperr"
)

type ErrRef string

func (e ErrRef) Error() string { return string(e) }

const ErrInvalidRef ErrRef = "invalid package reference"

type Ref struct {
	Name string
}

func Parse(raw string) (Ref, error) {
	name := strings.TrimSpace(raw)
	if name == "" || strings.HasPrefix(name, ".") || strings.Contains(name, ":") || filepath.IsAbs(name) {
		return Ref{}, apperr.AppErrMsgf(ErrInvalidRef, "invalid package ref %q", raw)
	}
	if strings.ContainsAny(name, `/\`) {
		return Ref{}, apperr.AppErrMsgf(ErrInvalidRef, "invalid package ref %q", raw)
	}
	if slices.Contains(strings.Split(name, "/"), "..") {
		return Ref{}, apperr.AppErrMsgf(ErrInvalidRef, "invalid package ref %q", raw)
	}
	return Ref{Name: name}, nil
}
