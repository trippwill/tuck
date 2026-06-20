package output

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

type Console struct {
	Invocation Invocation
	Color      bool
}

func NewConsole(inv Invocation, color bool) Console {
	return Console{
		Invocation: inv,
		Color:      color,
	}
}

func (c Console) Style(style Style, s string) string {
	if !c.Color || s == "" {
		return s
	}
	code := styleCode(style)
	if code == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func styleCode(style Style) string {
	switch style {
	case StyleAccent:
		return "1"
	case StyleMuted:
		return "2"
	case StyleSuccess:
		return "32"
	case StyleWarning:
		return "33"
	case StyleDanger:
		return "31;1"
	case StylePath:
		return "36"
	case StylePackage:
		return "35"
	default:
		return ""
	}
}
