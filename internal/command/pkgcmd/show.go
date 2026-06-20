package pkgcmd

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
)

const (
	CommandShow output.Command = "package show"
	KindTree    output.Kind    = "tree"
)

type ShowRequest struct {
	SourceID string
	Context  string
	Ref      string
}

func Show(req ShowRequest) output.Outcome {
	tree, err := packages.Show(packages.ShowOptions{
		SourceID: req.SourceID,
		Context:  req.Context,
		Ref:      req.Ref,
	})
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(output.Result{
		Kind:          KindTree,
		Data:          tree,
		ExitCode:      output.ExitOK,
		ConsoleString: renderTree,
	})
}

func renderTree(console output.Console, data any) (string, error) {
	p, ok := data.(packages.Tree)
	if !ok {
		return "", fmt.Errorf("package tree console renderer received %T", data)
	}
	inv := console.Invocation
	var b strings.Builder
	fmt.Fprintf(&b, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source)
	fmt.Fprintf(&b, "%s %s\n", console.Style(output.StyleAccent, "package:"), console.Style(output.StylePackage, p.Package.Identity))
	fmt.Fprintf(&b, "%s %s\n\n", console.Style(output.StyleAccent, "root:"), console.Style(output.StylePath, p.Package.Root))
	for _, entry := range p.Package.Entries {
		fmt.Fprintf(&b, "%s %s\n", console.Style(output.StyleMuted, fmt.Sprintf("%-4s", entry.Type)), entry.Rel)
	}
	if len(p.Package.Entries) > 0 {
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%s\n", console.Style(output.StyleMuted, fmt.Sprintf("%d %s", len(p.Package.Entries), entryNoun(len(p.Package.Entries)))))
	return b.String(), nil
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
