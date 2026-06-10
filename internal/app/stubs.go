package app

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func notImplemented(name string) cli.ActionFunc {
	return func(context.Context, *cli.Command) error {
		return cli.Exit(fmt.Sprintf("error: command %q is not implemented yet", name), ExitFail)
	}
}
