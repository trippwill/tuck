package pkgcmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/output"
)

const CommandDrop output.Command = "package drop"

type DropRequest struct {
	Refs     []string
	SourceID string
	Context  string
	Apply    bool
}

func Drop(req DropRequest) output.Outcome {
	planned, err := buildDrop(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}
