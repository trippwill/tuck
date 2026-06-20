package command

import (
	"errors"
	"testing"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/resolve"
)

func TestErrorResultMapsSentinels(t *testing.T) {
	tests := map[string]struct {
		err     error
		code    string
		message string
	}{
		"manifest": {
			err:     apperr.AppErrMsgf(manifest.ErrInvalid, "bad manifest"),
			code:    "manifest_invalid",
			message: "bad manifest",
		},
		"package": {
			err:     apperr.AppErrMsgf(packages.ErrPackageNotFound, "package %q not found", "zsh"),
			code:    "package_not_found",
			message: `package "zsh" not found`,
		},
		"plan": {
			err:     apperr.AppErrMsg(plan.ErrApply, "cannot apply a plan with conflicts"),
			code:    "io_error",
			message: "cannot apply a plan with conflicts",
		},
		"source": {
			err:     resolve.ErrUnknownSource,
			code:    "unknown_source",
			message: "source is not enabled or does not exist",
		},
		"fallback": {
			err:     errors.New("boom"),
			code:    "runtime_error",
			message: "boom",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ErrorResult(tt.err)
			if result.ExitCode != 1 {
				t.Fatalf("ExitCode = %d, want 1", result.ExitCode)
			}
			data, ok := result.Data.(output.ErrorData)
			if !ok {
				t.Fatalf("Result.Data = %T, want output.ErrorData", result.Data)
			}
			if data.Error.Code != tt.code || data.Error.Message != tt.message {
				t.Fatalf("ErrorResult() = (%q, %q), want (%q, %q)", data.Error.Code, data.Error.Message, tt.code, tt.message)
			}
		})
	}
}
