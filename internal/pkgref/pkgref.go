package pkgref

import (
	"path/filepath"
	"slices"
	"strings"
)

//go:generate go run ../../cmd/errgen -types ErrRef
type ErrRef string

const ErrInvalidRef ErrRef = "invalid package reference"

type Ref struct {
	Name string
}

func Parse(raw string) (Ref, error) {
	name := strings.TrimSpace(raw)
	if name == "" || name == "." || name == ".." || strings.Contains(name, ":") || filepath.IsAbs(name) {
		return Ref{}, AppErrMsgf(ErrInvalidRef, "invalid package ref %q", raw)
	}
	if strings.ContainsAny(name, `/\`) {
		return Ref{}, AppErrMsgf(ErrInvalidRef, "invalid package ref %q", raw)
	}
	if slices.Contains(strings.Split(name, "/"), "..") {
		return Ref{}, AppErrMsgf(ErrInvalidRef, "invalid package ref %q", raw)
	}
	return Ref{Name: name}, nil
}
