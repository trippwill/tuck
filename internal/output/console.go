package output

import "fmt"

type Style string

const (
	StyleAccent  Style = "accent"
	StyleMuted   Style = "muted"
	StyleSuccess Style = "success"
	StyleWarning Style = "warning"
	StyleDanger  Style = "danger"
	StylePath    Style = "path"
	StylePackage Style = "package"
)

type Palette map[Style]string

var DefaultPalette = Palette{
	StyleAccent:  "1",
	StyleMuted:   "2",
	StyleSuccess: "32",
	StyleWarning: "33",
	StyleDanger:  "31;1",
	StylePath:    "36",
	StylePackage: "35",
}

type Console struct {
	Invocation Invocation
	Color      bool
	Palette    Palette
}

func NewConsole(inv Invocation, color bool) Console {
	return Console{
		Invocation: inv,
		Color:      color,
		Palette:    DefaultPalette,
	}
}

func (c Console) Style(style Style, s string) string {
	if !c.Color || s == "" {
		return s
	}
	palette := c.Palette
	if palette == nil {
		palette = DefaultPalette
	}
	code := palette[style]
	if code == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (c Console) Sprintf(style Style, format string, args ...any) string {
	return c.Style(style, fmt.Sprintf(format, args...))
}
