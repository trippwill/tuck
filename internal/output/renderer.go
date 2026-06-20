package output

import (
	"encoding/json"
	"fmt"
	"io"
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

type Outcome struct {
	Payload Payload
	Err     error
}

func OK(payload Payload) Outcome {
	return Outcome{Payload: payload}
}

func Fail(err error) Outcome {
	return Outcome{Err: err}
}

func FailWith(payload Payload, err error) Outcome {
	return Outcome{Payload: payload, Err: err}
}

type Payload interface {
	Kind() Kind
	ExitCode() ExitCode
	JSONData() any
	WriteHuman(io.Writer, Invocation) error
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

type ErrorClassifier func(error) Error

type Options struct {
	Format        Format
	Color         bool
	Out           io.Writer
	Err           io.Writer
	ClassifyError ErrorClassifier
}

type Renderer struct {
	format        Format
	color         bool
	out           io.Writer
	err           io.Writer
	classifyError ErrorClassifier
}

func NewRenderer(options Options) Renderer {
	classifyError := options.ClassifyError
	if classifyError == nil {
		classifyError = fallbackError
	}
	return Renderer{
		format:        options.Format,
		color:         options.Color,
		out:           options.Out,
		err:           options.Err,
		classifyError: classifyError,
	}
}

func (r Renderer) Render(inv Invocation, outcome Outcome) (ExitCode, error) {
	if outcome.Err != nil {
		return r.renderError(inv, outcome)
	}
	if outcome.Payload == nil {
		return ExitOK, nil
	}
	return r.renderPayload(inv, outcome.Payload)
}

func (r Renderer) renderPayload(inv Invocation, payload Payload) (ExitCode, error) {
	exitCode := payload.ExitCode()
	if r.format == JSON {
		return exitCode, WriteEnvelope(r.out, inv.Command, inv.Context, payload.Kind(), payload.JSONData(), exitCode)
	}
	return exitCode, payload.WriteHuman(r.out, inv)
}

func (r Renderer) renderError(inv Invocation, outcome Outcome) (ExitCode, error) {
	appErr := r.classifyError(outcome.Err)
	if r.format == JSON {
		context := ""
		if outcome.Payload != nil {
			context = inv.Context
		}
		return ExitFail, WriteEnvelope(r.out, inv.Command, context, KindError, errorData{Error: appErr}, ExitFail)
	}
	if outcome.Payload != nil {
		if _, err := r.renderPayload(inv, outcome.Payload); err != nil {
			return ExitFail, err
		}
	}
	_, err := fmt.Fprintf(r.err, "error: %s\ncode: %s\nhint: %s\n", appErr.Message, appErr.Code, appErr.Hint)
	return ExitFail, err
}

type errorData struct {
	Error Error `json:"error"`
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

type InvalidArgsError struct {
	Message string
	Hint    string
}

func InvalidArgs(message string, hint string) InvalidArgsError {
	return InvalidArgsError{Message: message, Hint: hint}
}

func (e InvalidArgsError) Error() string {
	return e.Message
}
