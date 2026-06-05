//go:build tuck_testhooks

package state

import "testing"

func TestLoadUsesTestStateDirOverride(t *testing.T) {
	xdgRoot := t.TempDir()
	testRoot := t.TempDir()
	xdgRepo := writeSourceRepo(t, "xdg", "")
	testRepo := writeSourceRepo(t, "testhooks", "")
	writeSources(t, xdgRoot, sourceBlock("xdg", xdgRepo, nil))
	writeSources(t, testRoot, sourceBlock("testhooks", testRepo, nil))
	t.Setenv("XDG_STATE_HOME", xdgRoot)
	t.Setenv("TUCK_TEST_STATE_DIR", testRoot)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(got.Sources) != 1 || got.Sources[0].ID != "testhooks" {
		t.Fatalf("Load() sources = %#v, want only TUCK_TEST_STATE_DIR source", got.Sources)
	}
}
