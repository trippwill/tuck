package pathutil

import "testing"

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
