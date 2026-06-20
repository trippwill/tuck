package pkgcmd

import (
	"errors"

	"github.com/trippwill/tuck/internal/command/planout"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

const CommandRefresh output.Command = "package refresh"

type RefreshRequest struct {
	Refs     []string
	SourceID string
	Context  string
	Apply    bool
}

func Refresh(req RefreshRequest) output.Outcome {
	planned, err := plan.BuildRefresh(plan.RefreshOptions{
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
