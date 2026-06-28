package filecmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/command/statusout"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
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
	Copy     bool
	Mode     string
	SetMode  bool
	Replace  bool
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
	if req.SetMode && !req.Copy {
		return output.OK(output.InvalidArgs("adopt --mode requires --copy", "pass --copy with --mode"))
	}
	if req.SetMode {
		if _, err := packages.NormalizeModeFlag(req.Mode, "0644"); err != nil {
			return output.OK(output.InvalidArgs("mode must be octal or chmod-style rwx expression", "pass a mode like 0600 or u=rw,go="))
		}
	}
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
