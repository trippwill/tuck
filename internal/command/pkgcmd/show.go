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
	fmt.Fprintf(&b, "%s %s\n", console.Style(output.StyleAccent, "root:"), console.Style(output.StylePath, p.Package.Root))
	if len(p.Package.Entries) > 0 {
		fmt.Fprintln(&b)
		writeTree(&b, buildTree(p.Package.Entries), "")
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%s\n", console.Style(output.StyleMuted, fmt.Sprintf("%d %s", len(p.Package.Entries), entryNoun(len(p.Package.Entries)))))
	return b.String(), nil
}

type treeNode struct {
	name     string
	children []*treeNode
	index    map[string]*treeNode
}

func buildTree(entries []packages.TreeEntry) []*treeNode {
	root := &treeNode{}
	for _, entry := range entries {
		node := root
		for _, part := range strings.Split(entry.Rel, "/") {
			if part == "" {
				continue
			}
			if node.index == nil {
				node.index = make(map[string]*treeNode)
			}
			child := node.index[part]
			if child == nil {
				child = &treeNode{name: part}
				node.index[part] = child
				node.children = append(node.children, child)
			}
			node = child
		}
	}
	return root.children
}

func writeTree(b *strings.Builder, nodes []*treeNode, prefix string) {
	for i, node := range nodes {
		last := i == len(nodes)-1
		branch := "|-- "
		nextPrefix := prefix + "|   "
		if last {
			branch = "`-- "
			nextPrefix = prefix + "    "
		}
		fmt.Fprintf(b, "%s%s%s\n", prefix, branch, node.name)
		writeTree(b, node.children, nextPrefix)
	}
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
