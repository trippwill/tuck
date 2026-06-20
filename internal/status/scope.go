package status

import (
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/state"
)

func active(options Options) (state.Source, domain.TargetScope, error) {
	selection, err := domain.SelectActive(domain.SelectionOptions{
		SourceID:    options.SourceID,
		Context:     options.Context,
		RequireHome: false,
	})
	if err != nil {
		return state.Source{}, domain.TargetScope{}, err
	}
	return selection.Source, selection.Scope, nil
}

func expandPath(raw string) (string, error) { return pathutil.ExpandInput(raw) }
