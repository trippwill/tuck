package domain

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestTargetRoot(t *testing.T) {
	t.Run("missing home can be required", func(t *testing.T) {
		t.Setenv("HOME", "")

		_, err := targetRoot(true)
		if !errors.Is(err, ErrNoHome) {
			t.Fatalf("TargetRoot() error = %v, want errors.Is(..., ErrNoHome)", err)
		}
	})

	t.Run("missing home can default to current directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		got, err := targetRoot(false)
		if err != nil {
			t.Fatalf("TargetRoot() error = %v, want nil", err)
		}
		if got != "." {
			t.Fatalf("TargetRoot() = %q, want %q", got, ".")
		}
	})
}

func TestTargetScopePhysicalPath(t *testing.T) {
	t.Run("home uses logical path directly", func(t *testing.T) {
		t.Setenv("HOME", "/home/alice")
		scope, err := NewTargetScope(ContextHome, true)
		if err != nil {
			t.Fatalf("NewTargetScope() error = %v", err)
		}
		if got := scope.PhysicalPath("/home/alice/.zshrc"); got != "/home/alice/.zshrc" {
			t.Fatalf("PhysicalPath() = %q, want logical path", got)
		}
	})

	t.Run("root maps logical paths under physical root", func(t *testing.T) {
		scope := TargetScope{Context: ContextRoot, LogicalRoot: string(filepath.Separator), PhysicalRoot: "/tmp/root"}
		if got := scope.PhysicalPath("/etc/hosts"); got != "/tmp/root/etc/hosts" {
			t.Fatalf("PhysicalPath(/etc/hosts) = %q, want redirected root path", got)
		}
		if got := scope.PhysicalPath("/"); got != "/tmp/root" {
			t.Fatalf("PhysicalPath(/) = %q, want physical root", got)
		}
	})
}
