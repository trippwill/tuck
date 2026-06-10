package domain

import (
	"errors"
	"testing"
)

func TestTargetRoot(t *testing.T) {
	t.Run("explicit wins", func(t *testing.T) {
		t.Setenv("HOME", "/home/example")

		got, err := TargetRoot("/tmp/../target", true)
		if err != nil {
			t.Fatalf("TargetRoot() error = %v, want nil", err)
		}
		if got != "/target" {
			t.Fatalf("TargetRoot() = %q, want %q", got, "/target")
		}
	})

	t.Run("missing home can be required", func(t *testing.T) {
		t.Setenv("HOME", "")

		_, err := TargetRoot("", true)
		if !errors.Is(err, ErrNoHome) {
			t.Fatalf("TargetRoot() error = %v, want errors.Is(..., ErrNoHome)", err)
		}
	})

	t.Run("missing home can default to current directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		got, err := TargetRoot("", false)
		if err != nil {
			t.Fatalf("TargetRoot() error = %v, want nil", err)
		}
		if got != "." {
			t.Fatalf("TargetRoot() = %q, want %q", got, ".")
		}
	})
}
