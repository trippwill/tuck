package filecmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/output"
)

const CommandAdopt output.Command = "adopt"

type AdoptRequest struct {
	File     string
	Ref      string
	SourceID string
	Context  string
	Apply    bool
}

func Adopt(req AdoptRequest) output.Outcome {
	planned, err := buildAdopt(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}
