package app

import (
	"context"
	"os"
)

func Main() {
	if err := rootCommand().Run(context.Background(), os.Args); err != nil {
		os.Exit(ExitFail)
	}
}
