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

func TestConsoleStyleMissingPaletteEntryReturnsText(t *testing.T) {
	c := Console{Color: true, Palette: Palette{}}

	if got := c.Style(StyleSuccess, "ok"); got != "ok" {
		t.Fatalf("Style() = %q, want %q", got, "ok")
	}
}

func TestConsoleSprintfStylesFormattedText(t *testing.T) {
	c := Console{Color: true, Palette: Palette{StyleWarning: "33"}}

	if got := c.Sprintf(StyleWarning, "%s=%d", "count", 2); got != "\x1b[33mcount=2\x1b[0m" {
		t.Fatalf("Sprintf() = %q, want warning SGR with reset", got)
	}
}
