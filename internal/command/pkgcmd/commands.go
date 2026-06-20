package pkgcmd

import (
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/command/planconsole"
	"github.com/trippwill/tuck/internal/command/statusout"
	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
)

const (
	CommandUse     output.Command = "package use"
	CommandDrop    output.Command = "package drop"
	CommandRefresh output.Command = "package refresh"
	CommandStatus  output.Command = "package status"
)

type UseRequest struct {
	Refs     []string
	All      bool
	SourceID string
	Context  string
	Apply    bool
}

type DropRequest struct {
	Refs     []string
	SourceID string
	Context  string
	Apply    bool
}

type RefreshRequest struct {
	Refs     []string
	SourceID string
	Context  string
	Apply    bool
}

type StatusRequest struct {
	Ref      string
	SourceID string
	Context  string
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
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}

func Drop(req DropRequest) output.Outcome {
	planned, err := buildDrop(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
}

func Refresh(req RefreshRequest) output.Outcome {
	planned, err := buildRefresh(req)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(planconsole.Result(planned))
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
