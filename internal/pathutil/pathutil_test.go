package pathutil

import "testing"

func TestScaffold(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "unit test package is wired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {})
	}
}
