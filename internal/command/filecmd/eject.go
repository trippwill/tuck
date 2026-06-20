package filecmd

import (
	"errors"

	"github.com/trippwill/tuck/internal/command/planout"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

const CommandEject output.Command = "eject"

type EjectRequest struct {
	File     string
	SourceID string
	Context  string
	Apply    bool
}

func Eject(req EjectRequest) output.Outcome {
	planned, err := plan.BuildEject(plan.EjectOptions{
		File:     req.File,
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
