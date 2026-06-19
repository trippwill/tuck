package app

import (
	"github.com/trippwill/tuck/internal/domain"
	"github.com/urfave/cli/v3"
)

func domainFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "root", Usage: "use the root context (target /); default is home"},
		&cli.StringFlag{Name: "source", Aliases: []string{"s"}, Usage: "select active source by enabled id"},
	}
}

func mutatingDomainFlags() []cli.Flag {
	return append(domainFlags(), []cli.Flag{
		&cli.BoolFlag{Name: "apply", Usage: "execute the plan instead of just printing it"},
	}...)
}

func contextFromFlag(cmd *cli.Command) string {
	if cmd.Bool("root") {
		return domain.ContextRoot
	}
	return domain.ContextHome
}
