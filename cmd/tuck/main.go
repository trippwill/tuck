package main

import (
	"os"

	tuckcli "github.com/trippwill/tuck/internal/cli"
)

func main() {
	os.Exit(tuckcli.Run(os.Args, os.Environ(), os.Stdout, os.Stderr))
}
