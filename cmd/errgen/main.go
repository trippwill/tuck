package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/trippwill/tuck/cmd/errgen/internal/errgen"
)

func main() {
	typeName := flag.String("type", "", "sentinel error type name")
	constraintName := flag.String("constraint", "", "unsupported legacy generated constraint name")
	output := flag.String("output", "", "output file path")
	flag.Parse()

	if *output == "" && *typeName != "" {
		*output = "apperr_gen.go"
	}

	if _, err := errgen.Generate(errgen.Options{
		TypeName:       *typeName,
		ConstraintName: *constraintName,
		Dir:            ".",
		Output:         *output,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
