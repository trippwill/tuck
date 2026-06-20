package pkgcmd

import (
	"errors"

	"github.com/trippwill/tuck/internal/command/planout"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

const CommandUse output.Command = "package use"

type UseRequest struct {
	Refs     []string
	All      bool
	SourceID string
	Context  string
	Apply    bool
}

func Use(req UseRequest) output.Outcome {
	if len(req.Refs) == 0 && !req.All {
		return output.Fail(output.InvalidArgs(
			"package use requires one or more package refs or --all",
			"pass one or more package names, or use --all",
		))
	}
	if len(req.Refs) > 0 && req.All {
		return output.Fail(output.InvalidArgs(
			"package use accepts package refs or --all, not both",
			"choose package refs or --all",
		))
	}

	planned, err := plan.BuildUse(plan.UseOptions{
		Refs:     req.Refs,
		All:      req.All,
		SourceID: req.SourceID,
		Context:  req.Context,
		Apply:    req.Apply,
	})
	if err != nil {
		if errors.Is(err, plan.ErrPrivilegeRequired) {
			return output.FailWith(planout.FromPlan(planned), err)
		}
		return output.Fail(err)
	}
	return output.OK(planout.FromPlan(planned))
}
