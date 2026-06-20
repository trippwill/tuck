package statusout

import (
	"strings"
	"testing"

	"github.com/trippwill/tuck/internal/output"
)

func TestRenderStatusStylesStatesAndCodes(t *testing.T) {
	got, err := renderStatus(output.NewConsole(output.Invocation{Command: "status", Context: "home"}, true), Payload{
		Source: "public",
		Entries: []Entry{
			{TargetPath: "~/.zshrc", State: "deployed", Package: "source:home:zsh"},
			{TargetPath: "~/.config/nvim/init.lua", State: "conflict", Code: "target_exists"},
		},
	})
	if err != nil {
		t.Fatalf("renderStatus() error = %v", err)
	}
	if !strings.Contains(got, "\x1b[32mdeployed      \x1b[0m ~/.zshrc") {
		t.Fatalf("renderStatus() = %q, want styled deployed state", got)
	}
	if !strings.Contains(got, "\x1b[31;1mconflict      \x1b[0m ~/.config/nvim/init.lua code=\x1b[31;1mtarget_exists\x1b[0m") {
		t.Fatalf("renderStatus() = %q, want styled conflict state and code", got)
	}
}
