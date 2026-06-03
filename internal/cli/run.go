package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	urfavecli "github.com/urfave/cli/v3"
)

func Run(args []string, env []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"tuck"}
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	cmd := newCommand(stdout, stderr)
	err := cmd.Run(context.Background(), args)
	if err == nil {
		return ExitOK
	}

	var exitErr urfavecli.ExitCoder
	if errors.As(err, &exitErr) {
		if message := strings.TrimSpace(exitErr.Error()); message != "" {
			fmt.Fprintln(stderr, message)
		}
		return exitErr.ExitCode()
	}

	if message := strings.TrimSpace(err.Error()); message != "" {
		fmt.Fprintf(stderr, "error: %s\n", message)
	}
	return ExitUsage
}
