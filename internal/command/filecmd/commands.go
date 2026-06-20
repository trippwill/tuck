package filecmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/command/statusout"
	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
)

const (
	CommandAdopt  output.Command = "adopt"
	CommandEject  output.Command = "eject"
	CommandStatus output.Command = "status"
)

type AdoptRequest struct {
	File     string
	Ref      string
	SourceID string
	Context  string
	Apply    bool
}

type EjectRequest struct {
	File     string
	SourceID string
	Context  string
	Apply    bool
}

type StatusRequest struct {
	File     string
	SourceID string
	Context  string
}

func Adopt(req AdoptRequest) output.Outcome {
	planned, err := buildAdopt(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}

func Eject(req EjectRequest) output.Outcome {
	planned, err := buildEject(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}

func Status(req StatusRequest) output.Outcome {
	result, err := statuspkg.File(req.File, statuspkg.Options{
		SourceID: req.SourceID,
		Context:  req.Context,
	})
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(statusout.FromResult(result))
}
