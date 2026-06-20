package domain

import (
	"errors"
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
