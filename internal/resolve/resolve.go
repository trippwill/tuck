package resolve

import (
	"errors"

	"github.com/trippwill/tuck/internal/state"
)

type ErrSource string

const (
	ErrNoSource      ErrSource = "no source"
	ErrUnknownSource ErrSource = "unknown source"
)

func (e ErrSource) Error() string {
	return string(e)
}

func ActiveSource(registry state.Registry, explicitID string) (state.Source, error) {
	return state.Source{}, errors.New("active source resolution not implemented")
}
