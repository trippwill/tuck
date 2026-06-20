package pkgcmd

import (
	"fmt"
	"strings"

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

type TreePayload struct {
	Source  string      `json:"-"`
	Package TreePackage `json:"package"`
}

type TreePackage struct {
	Identity string      `json:"identity"`
	Root     string      `json:"root"`
	Entries  []TreeEntry `json:"entries"`
}

type TreeEntry struct {
	Rel  string `json:"rel"`
	Type string `json:"type"`
}

func Show(req ShowRequest) output.Outcome {
	tree, err := packages.Show(packages.ShowOptions{
		SourceID: req.SourceID,
		Context:  req.Context,
		Ref:      req.Ref,
	})
	if err != nil {
		return errorOutcome(err)
	}
	payload := TreePayload{
		Source: tree.Source,
		Package: TreePackage{
			Identity: tree.Package.Identity,
			Root:     tree.Package.Root,
			Entries:  fromTreeEntries(tree.Package.Entries),
		},
	}
	return output.OK(output.Result{
		Kind:          KindTree,
		Data:          payload,
		ExitCode:      output.ExitOK,
		ConsoleString: renderTree,
	})
}

func renderTree(inv output.Invocation, data any) (string, error) {
	p, ok := data.(TreePayload)
	if !ok {
		return "", fmt.Errorf("package tree console renderer received %T", data)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source)
	fmt.Fprintf(&b, "package: %s\n", p.Package.Identity)
	fmt.Fprintf(&b, "root: %s\n\n", p.Package.Root)
	for _, entry := range p.Package.Entries {
		fmt.Fprintf(&b, "%-4s %s\n", entry.Type, entry.Rel)
	}
	if len(p.Package.Entries) > 0 {
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%d %s\n", len(p.Package.Entries), entryNoun(len(p.Package.Entries)))
	return b.String(), nil
}

func fromTreeEntries(entries []packages.TreeEntry) []TreeEntry {
	out := make([]TreeEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, TreeEntry{Rel: entry.Rel, Type: entry.Type})
	}
	return out
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
