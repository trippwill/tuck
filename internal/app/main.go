package app

import (
	"context"
	"os"
)

func Main() {
	if handled, err := runMetaJSON(os.Args, os.Stdout); handled {
		if err != nil {
			os.Exit(ExitFail)
		}
		return
	}
	if err := rootCommand().Run(context.Background(), os.Args); err != nil {
		os.Exit(ExitFail)
	}
}
