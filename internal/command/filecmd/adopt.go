package filecmd

import (
	"errors"

	"github.com/trippwill/tuck/internal/command/planout"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
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
	planned, err := plan.BuildAdopt(plan.AdoptOptions{
		File:     req.File,
		Ref:      req.Ref,
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
