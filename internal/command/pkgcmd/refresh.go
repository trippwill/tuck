package pkgcmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/output"
)

const CommandRefresh output.Command = "package refresh"

type RefreshRequest struct {
	Refs     []string
	SourceID string
	Context  string
	Apply    bool
}

func Refresh(req RefreshRequest) output.Outcome {
	planned, err := buildRefresh(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}
