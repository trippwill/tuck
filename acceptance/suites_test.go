//go:build tuck_testhooks

package acceptance

import "testing"

func TestSuites(t *testing.T) {
	tests := []struct {
		name  string
		suite string
	}{
		{name: "foundation", suite: "foundation"},
		{name: "json", suite: "json"},
		{name: "package", suite: "package"},
		{name: "source", suite: "source"},
		{name: "target", suite: "target"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runScriptSuite(t, tt.suite)
		})
	}
}
