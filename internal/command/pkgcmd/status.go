package pkgcmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/statusout"
	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
)

const CommandStatus output.Command = "package status"

type StatusRequest struct {
	Ref      string
	SourceID string
	Context  string
}

func Status(req StatusRequest) output.Outcome {
	result, err := statuspkg.Package(req.Ref, statuspkg.Options{
		SourceID: req.SourceID,
		Context:  req.Context,
	})
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(statusout.FromResult(result))
}
