package packages

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/state"
)

func TestShowBuildsDeterministicPackageTree(t *testing.T) {
	source := setupShowSource(t)
	writeShowFile(t, filepath.Join(source, "zsh/.config/zsh/.zshrc"))
	writeShowFile(t, filepath.Join(source, "zsh/.config/zsh/.zprofile"))

	got, err := Show(ShowOptions{Ref: "zsh"})
	if err != nil {
		t.Fatalf("Show() error = %v, want nil", err)
	}
	if got.Command != "package show" || got.Context != ContextHome || got.Source != "public" {
		t.Fatalf("Show() metadata = %#v, want package show in home/public", got)
	}
	if got.Package.Identity != "public:home:zsh" || got.Package.Root != filepath.Join(source, "zsh") {
		t.Fatalf("Show() package = %#v, want public:home:zsh rooted in source", got.Package)
	}
	want := []TreeEntry{
		{Rel: ".config", Type: "dir"},
		{Rel: ".config/zsh", Type: "dir"},
		{Rel: ".config/zsh/.zprofile", Type: "leaf"},
		{Rel: ".config/zsh/.zshrc", Type: "leaf"},
	}
	if !reflect.DeepEqual(got.Package.Entries, want) {
		t.Fatalf("Show() entries = %#v, want %#v", got.Package.Entries, want)
	}
}

func TestShowUsesRootContextPackageBase(t *testing.T) {
	source := setupShowSource(t)
	writeShowFile(t, filepath.Join(source, ".root/sshd/etc/ssh/sshd_config"))

	got, err := Show(ShowOptions{Context: ContextRoot, Ref: "sshd"})
	if err != nil {
		t.Fatalf("Show() error = %v, want nil", err)
	}
	if got.Context != ContextRoot || got.Package.Identity != "public:root:sshd" {
		t.Fatalf("Show() = %#v, want root sshd package", got)
	}
	want := []TreeEntry{
		{Rel: "etc", Type: "dir"},
		{Rel: "etc/ssh", Type: "dir"},
		{Rel: "etc/ssh/sshd_config", Type: "leaf"},
	}
	if !reflect.DeepEqual(got.Package.Entries, want) {
		t.Fatalf("Show() entries = %#v, want %#v", got.Package.Entries, want)
	}
}

func TestShowIncludesDeployMetadataForHumanRendering(t *testing.T) {
	source := setupShowSource(t)
	writeShowFile(t, filepath.Join(source, "app/.config/app/config"))
	writeShowFile(t, filepath.Join(source, "app/.tuck.toml"), `[[file]]
path = ".config/app/config"
deploy = "copy"
`)

	got, err := Show(ShowOptions{Ref: "app"})
	if err != nil {
		t.Fatalf("Show() error = %v, want nil", err)
	}
	want := []TreeEntry{
		{Rel: ".config", Type: "dir"},
		{Rel: ".config/app", Type: "dir"},
		{Rel: ".config/app/config", Type: "leaf", Deploy: DeployCopy},
	}
	if !reflect.DeepEqual(got.Package.Entries, want) {
		t.Fatalf("Show() entries = %#v, want %#v", got.Package.Entries, want)
	}
}

func setupShowSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "src")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	writeShowFile(t, filepath.Join(source, manifest.ManifestFilename), "name = \"public\"\n")
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if err := state.Save(state.Registry{
		Default: "public",
		Sources: []state.Source{{
			ID:      "public",
			Path:    source,
			Enabled: true,
		}},
	}); err != nil {
		t.Fatalf("state.Save() error = %v", err)
	}
	return source
}

func writeShowFile(t *testing.T, path string, contents ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "contents"
	if len(contents) > 0 {
		body = contents[0]
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
