package pkgcmd

import (
	"errors"

	"github.com/trippwill/tuck/internal/command/planout"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

const CommandDrop output.Command = "package drop"

type DropRequest struct {
	Refs     []string
	SourceID string
	Context  string
	Apply    bool
}

func Drop(req DropRequest) output.Outcome {
	planned, err := plan.BuildDrop(plan.DropOptions{
		Refs:     req.Refs,
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
