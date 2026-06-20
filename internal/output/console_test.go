package output

import "testing"

func TestConsoleStyleDisabledReturnsText(t *testing.T) {
	c := NewConsole(Invocation{Command: "test"}, false)

	if got := c.Style(StyleSuccess, "ok"); got != "ok" {
		t.Fatalf("Style() = %q, want %q", got, "ok")
	}
}

func TestConsoleStyleEnabledWrapsAndResetsText(t *testing.T) {
	c := NewConsole(Invocation{Command: "test"}, true)

	if got := c.Style(StyleSuccess, "ok"); got != "\x1b[32mok\x1b[0m" {
		t.Fatalf("Style() = %q, want success SGR with reset", got)
	}
}

func TestConsoleStyleUnknownStyleReturnsText(t *testing.T) {
	c := NewConsole(Invocation{Command: "test"}, true)

	if got := c.Style(Style("unknown"), "ok"); got != "ok" {
		t.Fatalf("Style() = %q, want %q", got, "ok")
	}
}
