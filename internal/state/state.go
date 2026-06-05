package state

import (
	"errors"

	"github.com/trippwill/tuck/internal/manifest"
)

type Source struct {
	ID       string
	Path     string
	Enabled  bool
	Manifest manifest.Manifest
}

type Registry struct {
	Sources []Source
	Default string
}

type ErrState string

const ErrInvalid ErrState = "invalid state"

func (e ErrState) Error() string {
	return string(e)
}

func Load() (Registry, error) {
	return Registry{}, errors.New("state load not implemented")
}
