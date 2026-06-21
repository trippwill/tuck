package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
	"github.com/urfave/cli/v3"
)

func completePackages(ctx context.Context, cmd *cli.Command) {
	if completeFlags(ctx, cmd) {
		return
	}
	completePackageCandidates(cmd)
}

func completeFirstPackage(ctx context.Context, cmd *cli.Command) {
	if completeFlags(ctx, cmd) {
		return
	}
	if len(cmd.Args().Slice()) > 1 {
		return
	}
	completePackageCandidates(cmd)
}

func completePackageCandidates(cmd *cli.Command) {
	listing, err := packages.List(packages.ListOptions{
		SourceID: cmd.String("source"),
		Context:  contextFromFlag(cmd),
	})
	if err != nil {
		// ponytail: completions are best-effort; real commands surface state errors.
		return
	}
	printCompletions(cmd, listing.Packages)
}

func completeEnabledSources(ctx context.Context, cmd *cli.Command) {
	completeSources(ctx, cmd, true)
}

func completeAllSources(ctx context.Context, cmd *cli.Command) {
	completeSources(ctx, cmd, false)
}

func completeSources(ctx context.Context, cmd *cli.Command, enabledOnly bool) {
	if completeFlags(ctx, cmd) {
		return
	}
	registry, err := state.Load()
	if err != nil {
		// ponytail: completions are best-effort; real commands surface state errors.
		return
	}
	ids := make([]string, 0, len(registry.Sources))
	for _, source := range registry.Sources {
		if enabledOnly && !source.Enabled {
			continue
		}
		ids = append(ids, source.ID)
	}
	printCompletions(cmd, ids)
}

func completeFlags(ctx context.Context, cmd *cli.Command) bool {
	if strings.HasPrefix(completionPrefix(cmd), "-") {
		cli.DefaultCompleteWithFlags(ctx, cmd)
		return true
	}
	return false
}

func printCompletions(cmd *cli.Command, candidates []string) {
	prefix := completionPrefix(cmd)
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			fmt.Fprintln(cmd.Root().Writer, candidate)
		}
	}
}

func completionPrefix(cmd *cli.Command) string {
	args := cmd.Args().Slice()
	if len(args) == 0 {
		return ""
	}
	return args[len(args)-1]
}
