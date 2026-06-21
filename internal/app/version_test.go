package app

import "testing"

func TestRootCommandUsesStampedVersion(t *testing.T) {
	oldVersion := version
	t.Cleanup(func() { version = oldVersion })
	version = "v1.2.3"

	if got := rootCommand().Version; got != "v1.2.3" {
		t.Fatalf("rootCommand().Version = %q, want stamped version", got)
	}
}
