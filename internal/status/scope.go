package status

import (
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/state"
)

func active(options Options) (state.Source, domain.TargetScope, error) {
	context := packages.ContextHome
	if options.Context == packages.ContextRoot {
		context = packages.ContextRoot
	}
	scope, err := domain.NewTargetScope(context, options.TargetRoot, false)
	if err != nil {
		return state.Source{}, domain.TargetScope{}, err
	}
	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return state.Source{}, domain.TargetScope{}, err
	}
	return source, scope, nil
}

func expandPath(raw string) (string, error) { return pathutil.ExpandInput(raw) }
