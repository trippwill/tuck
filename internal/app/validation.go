package app

import "github.com/urfave/cli/v3"

func noExtraArgs(cmd *cli.Command, command string, message string, hint string) error {
	return invalidArgsIf(cmd.Args().Present(), cmd, command, message, hint)
}

func invalidArgsIf(condition bool, cmd *cli.Command, command string, message string, hint string) error {
	if !condition {
		return nil
	}
	return renderInvalidArgs(cmd, command, message, hint)
}
