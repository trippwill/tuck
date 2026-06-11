package pathutil

import (
	"path/filepath"
	"testing"
)

func TestInsideIsPathSegmentAware(t *testing.T) {
	if !Inside("/home/me/.dotfiles/zsh", "/home/me/.dotfiles") {
		t.Fatalf("Inside() = false, want true")
	}
	if Inside("/home/me/.dotfiles-private", "/home/me/.dotfiles") {
		t.Fatalf("Inside() = true for sibling prefix, want false")
	}
}

func TestPackageToTarget(t *testing.T) {
	got, rel, err := PackageToTarget("/src/zsh", "/src/zsh/.config/zsh/.zshrc", "/home/me")
	if err != nil {
		t.Fatalf("PackageToTarget() error = %v", err)
	}
	if got != "/home/me/.config/zsh/.zshrc" || rel != ".config/zsh/.zshrc" {
		t.Fatalf("PackageToTarget() = %q, %q", got, rel)
	}
}

func TestSymlinkPayload(t *testing.T) {
	got, err := SymlinkPayload("/home/me/.config/zsh/.zshrc", "/src/zsh/.config/zsh/.zshrc")
	if err != nil {
		t.Fatalf("SymlinkPayload() error = %v", err)
	}
	if got != "../../../../src/zsh/.config/zsh/.zshrc" {
		t.Fatalf("SymlinkPayload() = %q", got)
	}
}

func TestExpandInput(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Chdir(filepath.Join(root))

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "home", raw: "~", want: home},
		{name: "home child", raw: "~/dot/../file", want: filepath.Join(home, "file")},
		{name: "relative", raw: "relative/../file", want: filepath.Join(root, "file")},
		{name: "absolute", raw: filepath.Join(root, "abs/../file"), want: filepath.Join(root, "file")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandInput(tt.raw)
			if err != nil {
				t.Fatalf("ExpandInput() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ExpandInput() = %q, want %q", got, tt.want)
			}
		})
	}
}
