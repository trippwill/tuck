package app

import (
	"fmt"

	statuspkg "github.com/trippwill/tuck/internal/status"
	"github.com/urfave/cli/v3"
)

func renderStatus(cmd *cli.Command, result statuspkg.Result) error {
	r := newRenderer(cmd)
	if r.json {
		return r.writeEnvelope(result.Command, result.Context, "status", result, ExitOK)
	}

	fmt.Fprintf(r.out, "tuck %s   (context: %s, source: %s)\n\n", result.Command, result.Context, result.Source)
	for _, entry := range result.Entries {
		fmt.Fprintf(r.out, "%-14s %s", entry.State, entry.TargetPath)
		if entry.Package != "" {
			fmt.Fprintf(r.out, " package=%s", entry.Package)
		}
		if entry.Entry != "" {
			fmt.Fprintf(r.out, " entry=%s", entry.Entry)
		}
		if entry.Owner != "" && entry.Owner != entry.Package {
			fmt.Fprintf(r.out, " owner=%s", entry.Owner)
		}
		if entry.Code != "" {
			fmt.Fprintf(r.out, " code=%s", entry.Code)
		}
		if entry.Message != "" {
			fmt.Fprintf(r.out, " (%s)", entry.Message)
		}
		fmt.Fprintln(r.out)
	}
	fmt.Fprintf(r.out, "\n%d %s\n", len(result.Entries), entryNoun(len(result.Entries)))
	return nil
}
