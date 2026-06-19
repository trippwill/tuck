package app

import (
	"fmt"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/urfave/cli/v3"
)

type packageListData struct {
	Source   string   `json:"source"`
	Packages []string `json:"packages"`
}

func renderPackageList(cmd *cli.Command, listing packages.Listing) error {
	r := newRenderer(cmd)
	data := packageListData{Source: listing.Source, Packages: listing.Packages}
	if r.json {
		return r.writeEnvelope("package list", listing.Context, "packages", data, ExitOK)
	}

	fmt.Fprintf(r.out, "tuck package list   (context: %s, source: %s)\n\n", listing.Context, listing.Source)
	for _, name := range listing.Packages {
		fmt.Fprintln(r.out, name)
	}
	if len(listing.Packages) > 0 {
		fmt.Fprintln(r.out)
	}
	fmt.Fprintf(r.out, "%d %s\n", len(listing.Packages), packageNoun(len(listing.Packages)))
	return nil
}

func renderPackageTree(cmd *cli.Command, tree packages.Tree) error {
	r := newRenderer(cmd)
	if r.json {
		return r.writeEnvelope(tree.Command, tree.Context, "tree", tree, ExitOK)
	}

	fmt.Fprintf(r.out, "tuck package show   (context: %s, source: %s)\n\n", tree.Context, tree.Source)
	fmt.Fprintf(r.out, "package: %s\n", tree.Package.Identity)
	fmt.Fprintf(r.out, "root: %s\n\n", tree.Package.Root)
	for _, entry := range tree.Package.Entries {
		fmt.Fprintf(r.out, "%-4s %s\n", entry.Type, entry.Rel)
	}
	if len(tree.Package.Entries) > 0 {
		fmt.Fprintln(r.out)
	}
	fmt.Fprintf(r.out, "%d %s\n", len(tree.Package.Entries), entryNoun(len(tree.Package.Entries)))
	return nil
}

func packageNoun(count int) string {
	if count == 1 {
		return "package"
	}
	return "packages"
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}
