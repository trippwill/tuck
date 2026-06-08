package pkgref

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "zsh", want: "zsh", ok: true},
		{input: " git ", want: "git", ok: true},
		{input: "", ok: false},
		{input: "a:b", ok: false},
		{input: "/zsh", ok: false},
		{input: "a/b", ok: false},
		{input: `a\b`, ok: false},
		{input: "..", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.ok {
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				if got.Name != tt.want {
					t.Fatalf("Parse().Name = %q, want %q", got.Name, tt.want)
				}
				return
			}
			if !errors.Is(err, ErrInvalidRef) {
				t.Fatalf("Parse() error = %v, want ErrInvalidRef", err)
			}
		})
	}
}
