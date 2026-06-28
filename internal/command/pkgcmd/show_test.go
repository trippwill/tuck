package pkgcmd

import (
	"testing"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
)

func TestRenderTreeUsesASCIIConnectors(t *testing.T) {
	got, err := renderTree(output.NewConsole(output.Invocation{Command: "package show", Context: "home"}, false), packages.Tree{
		Source: "public",
		Package: packages.TreePackage{
			Identity: "public:home:zsh",
			Root:     "/src/zsh",
			Entries: []packages.TreeEntry{
				{Rel: ".config", Type: "dir"},
				{Rel: ".config/zsh", Type: "dir"},
				{Rel: ".config/zsh/.zlogin", Type: "leaf"},
				{Rel: ".config/zsh/.zshrc", Type: "leaf"},
			},
		},
	})
	if err != nil {
		t.Fatalf("renderTree() error = %v", err)
	}
	want := "tuck package show   (context: home, source: public)\n\n" +
		"package: public:home:zsh\n" +
		"root: /src/zsh\n\n" +
		"`-- .config\n" +
		"    `-- zsh\n" +
		"        |-- .zlogin\n" +
		"        `-- .zshrc\n\n" +
		"4 entries\n"
	if got != want {
		t.Fatalf("renderTree() = %q, want %q", got, want)
	}
}
