package app

import (
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

func mutatingFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "apply", Usage: "execute the plan instead of just printing it"},
	}
}

func contextFromFlag(cmd *cli.Command) string {
	if cmd.Bool("root") {
		return domain.ContextRoot
	}
	return domain.ContextHome
}

func ignoredDomainSelectionWarnings(cmd *cli.Command, command output.Command) []output.Warning {
	warnings := make([]output.Warning, 0, 2)
	if cmd.IsSet("source") {
		warnings = append(warnings, output.Warning{
			Code:    "ignored_flag",
			Message: string(command) + " ignores --source",
			Hint:    "source commands manage the registry and do not select an active source",
		})
	}
	if cmd.IsSet("root") {
		warnings = append(warnings, output.Warning{
			Code:    "ignored_flag",
			Message: string(command) + " ignores --root",
			Hint:    "source commands manage the registry and do not select a target context",
		})
	}
	return warnings
}
