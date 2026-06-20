package resolve

import "github.com/trippwill/tuck/internal/state"

type ErrSource string

func (e ErrSource) Error() string { return string(e) }

const (
	ErrNoSource      ErrSource = "no source"
	ErrUnknownSource ErrSource = "unknown source"
)

func ActiveSource(registry state.Registry, explicitID string) (state.Source, error) {
	enabledSources := registry.EnabledSources()

	if explicitID != "" {
		for _, source := range enabledSources {
			if source.ID == explicitID {
				return source, nil
			}
		}
		return state.Source{}, ErrUnknownSource
	}

	if registry.Default != "" {
		for _, source := range enabledSources {
			if source.ID == registry.Default {
				return source, nil
			}
		}
		return state.Source{}, ErrNoSource
	}

	if len(enabledSources) == 1 {
		return enabledSources[0], nil
	}

	return state.Source{}, ErrNoSource
}
