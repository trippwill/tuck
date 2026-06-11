package status

import (
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/state"
)

func active(options Options) (state.Source, string, error) {
	targetRoot, err := domain.TargetRoot(options.TargetRoot, false)
	if err != nil {
		return state.Source{}, "", err
	}
	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return state.Source{}, "", err
	}
	return source, targetRoot, nil
}

func expandPath(raw string) (string, error) { return pathutil.ExpandInput(raw) }
