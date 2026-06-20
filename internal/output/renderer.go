package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Format uint8

const (
	Human Format = iota
	JSON
)

type ExitCode int

const (
	ExitOK   ExitCode = 0
	ExitFail ExitCode = 1
)

type Kind string

const KindError Kind = "error"

type Command string

type Invocation struct {
	Command Command
	Context string
}

type ConsoleStringFunc func(Console, any) (string, error)

type Result struct {
	Kind          Kind
	Data          any
	ExitCode      ExitCode
	ConsoleString ConsoleStringFunc
}

type Outcome struct {
	Result *Result
	Err    error
}

func OK(value any) Outcome {
	switch v := value.(type) {
	case Result:
		return Outcome{Result: &v}
	case *Result:
		return Outcome{Result: v}
	default:
		return Outcome{Err: fmt.Errorf("unsupported output result %T", value)}
	}
}

func Fail(err error) Outcome {
	return Outcome{Err: err}
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

type ErrorData struct {
	Error Error `json:"error"`
}

func ErrorResult(code, message, hint string) Result {
	record := Error{Code: code, Message: message, Hint: hint}
	return Result{
		Kind:     KindError,
		Data:     ErrorData{Error: record},
		ExitCode: ExitFail,
		ConsoleString: func(console Console, _ any) (string, error) {
			return FormatConsoleError(console, record), nil
		},
	}
}

func InvalidArgs(message string, hint string) Result {
	return ErrorResult("invalid_args", message, hint)
}

func IOError(message string) Result {
	return ErrorResult("io_error", message, "retry after fixing filesystem permissions or disk state")
}

func DetailMessage(err error, fallback string, sentinels ...error) string {
	if err == nil {
		return fallback
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fallback
	}
	for _, sentinel := range sentinels {
		if sentinel == nil {
			continue
		}
		sentinelText := sentinel.Error()
		if message == sentinelText {
			return fallback
		}
		if after, ok := strings.CutPrefix(message, sentinelText+": "); ok && after != "" {
			return after
		}
	}
	return message
}

func FormatError(record Error) string {
	return FormatConsoleError(Console{}, record)
}

func FormatConsoleError(console Console, record Error) string {
	return fmt.Sprintf("%s %s\n%s %s\n%s %s\n",
		console.Style(StyleDanger, "error:"),
		record.Message,
		console.Style(StyleMuted, "code:"),
		console.Style(StyleMuted, record.Code),
		console.Style(StyleWarning, "hint:"),
		record.Hint,
	)
}

type Options struct {
	Format   Format
	Color    bool
	ErrColor bool
	Out      io.Writer
	Err      io.Writer
}

type Renderer struct {
	format   Format
	color    bool
	errColor bool
	out      io.Writer
	err      io.Writer
}

func NewRenderer(options Options) Renderer {
	return Renderer{
		format:   options.Format,
		color:    options.Color,
		errColor: options.ErrColor,
		out:      options.Out,
		err:      options.Err,
	}
}

func (r Renderer) Render(inv Invocation, outcome Outcome) (ExitCode, error) {
	if outcome.Err != nil {
		return r.renderError(inv, outcome)
	}
	if outcome.Result != nil {
		return r.renderResult(inv, *outcome.Result)
	}
	return ExitOK, nil
}

func (r Renderer) renderResult(inv Invocation, result Result) (ExitCode, error) {
	exitCode := result.ExitCode
	if r.format == JSON {
		context := inv.Context
		if result.Kind == KindError {
			context = ""
		}
		return exitCode, WriteEnvelope(r.out, inv.Command, context, result.Kind, result.Data, exitCode)
	}
	if result.ConsoleString == nil {
		return ExitFail, fmt.Errorf("missing console renderer for %q result", result.Kind)
	}
	console := NewConsole(inv, r.color)
	if result.Kind == KindError {
		console.Color = r.errColor
	}
	text, err := result.ConsoleString(console, result.Data)
	if err != nil {
		return ExitFail, err
	}
	w := r.out
	if result.Kind == KindError {
		w = r.err
	}
	_, err = io.WriteString(w, text)
	return exitCode, err
}

func (r Renderer) renderError(inv Invocation, outcome Outcome) (ExitCode, error) {
	appErr := fallbackError(outcome.Err)
	if r.format == JSON {
		return ExitFail, WriteEnvelope(r.out, inv.Command, "", KindError, ErrorData{Error: appErr}, ExitFail)
	}
	_, err := io.WriteString(r.err, FormatConsoleError(NewConsole(inv, r.errColor), appErr))
	return ExitFail, err
}

type envelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Context       string `json:"context,omitempty"`
	Kind          string `json:"kind"`
	Data          any    `json:"data"`
	ExitCode      int    `json:"exitCode"`
}

func WriteEnvelope(out io.Writer, command Command, context string, kind Kind, data any, exitCode ExitCode) error {
	return writeJSON(out, envelope{
		SchemaVersion: 1,
		Command:       string(command),
		Context:       context,
		Kind:          string(kind),
		Data:          data,
		ExitCode:      int(exitCode),
	})
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func fallbackError(err error) Error {
	message := "runtime error"
	if err != nil && err.Error() != "" {
		message = err.Error()
	}
	return Error{
		Code:    "io_error",
		Message: message,
		Hint:    "retry after fixing filesystem permissions or disk state",
	}
}
