package filecmd

import (
	"github.com/trippwill/tuck/internal/command/statusout"
	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
)

const CommandStatus output.Command = "status"

type StatusRequest struct {
	File     string
	SourceID string
	Context  string
}

func Status(req StatusRequest) output.Outcome {
	result, err := statuspkg.File(req.File, statuspkg.Options{
		SourceID: req.SourceID,
		Context:  req.Context,
	})
	if err != nil {
		return output.Fail(err)
	}
	return output.OK(statusout.FromResult(result))
}
