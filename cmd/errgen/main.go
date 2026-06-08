package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/trippwill/tuck/cmd/errgen/internal/errgen"
)

func main() {
	typeNames := flag.String("types", "", "comma-separated sentinel error type names")
	constraintName := flag.String("constraint", "", "generated constraint name")
	output := flag.String("output", "", "output file path")
	flag.Parse()

	if *output == "" && *typeNames != "" {
		*output = "apperr_gen.go"
	}

	if _, err := errgen.Generate(errgen.Options{
		TypeNames:      splitTypes(*typeNames),
		ConstraintName: *constraintName,
		Dir:            ".",
		Output:         *output,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func splitTypes(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	types := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			types = append(types, trimmed)
		}
	}
	return types
}
