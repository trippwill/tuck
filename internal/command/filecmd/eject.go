package filecmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/output"
)

const CommandEject output.Command = "eject"

type EjectRequest struct {
	File       string
	SourceID   string
	Context    string
	TargetRoot string
	Apply      bool
}

func Eject(req EjectRequest) output.Outcome {
	planned, err := buildEject(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}
