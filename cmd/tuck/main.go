package main

import (
	"os"

	"github.com/trippwill/tuck/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args, os.Environ(), os.Stdout, os.Stderr))
}
