package pkgcmd

import (
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
)

func TestRenderTreeUsesUnicodeConnectorsByDefault(t *testing.T) {
	got, err := renderTree(output.NewConsole(output.Invocation{Command: "package show", Context: "home"}, false, false), packages.Tree{
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
		"└── .config\n" +
		"    └── zsh\n" +
		"        ├── .zlogin\n" +
		"        └── .zshrc\n\n" +
		"4 entries\n"
	if got != want {
		t.Fatalf("renderTree() = %q, want %q", got, want)
	}
}

func TestRenderTreeLabelsCopyEntries(t *testing.T) {
	got, err := renderTree(output.NewConsole(output.Invocation{Command: "package show", Context: "home"}, false, true), packages.Tree{
		Source: "public",
		Package: packages.TreePackage{
			Identity: "public:home:app",
			Root:     "/src/app",
			Entries: []packages.TreeEntry{
				{Rel: ".config", Type: "dir"},
				{Rel: ".config/app", Type: "dir"},
				{Rel: ".config/app/config", Type: "leaf", Deploy: packages.DeployCopy},
			},
		},
	})
	if err != nil {
		t.Fatalf("renderTree() error = %v", err)
	}
	want := "tuck package show   (context: home, source: public)\n\n" +
		"package: public:home:app\n" +
		"root: /src/app\n\n" +
		"key: [copy] deploy=copy\n\n" +
		"`-- .config\n" +
		"    `-- app\n" +
		"        `-- config [copy]\n\n" +
		"3 entries\n"
	if got != want {
		t.Fatalf("renderTree() = %q, want %q", got, want)
	}
}

func TestRenderTreeUsesASCIIWhenConsoleRequestsIt(t *testing.T) {
	got, err := renderTree(output.NewConsole(output.Invocation{Command: "package show", Context: "home"}, false, true), packages.Tree{
		Source:  "public",
		Package: packages.TreePackage{Entries: []packages.TreeEntry{{Rel: "file", Type: "leaf"}}},
	})
	if err != nil {
		t.Fatalf("renderTree() error = %v", err)
	}
	if !strings.Contains(got, "\n`-- file\n") {
		t.Fatalf("renderTree() = %q, want ASCII tree connector", got)
	}
}
