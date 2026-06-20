package pkgcmd

import (
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/output"
)

const CommandUse output.Command = "package use"

type UseRequest struct {
	Refs       []string
	All        bool
	SourceID   string
	Context    string
	TargetRoot string
	Apply      bool
}

func Use(req UseRequest) output.Outcome {
	if len(req.Refs) == 0 && !req.All {
		return output.OK(output.InvalidArgs(
			"package use requires one or more package refs or --all",
			"pass one or more package names, or use --all",
		))
	}
	if len(req.Refs) > 0 && req.All {
		return output.OK(output.InvalidArgs(
			"package use accepts package refs or --all, not both",
			"choose package refs or --all",
		))
	}

	planned, err := buildUse(req)
	if err != nil {
		return errorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}
