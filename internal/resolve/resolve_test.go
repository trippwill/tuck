package resolve

import (
	"errors"
	"testing"

	"github.com/trippwill/tuck/internal/state"
)

func TestActiveSource(t *testing.T) {
	tests := []struct {
		name       string
		registry   state.Registry
		explicitID string
		wantID     string
		wantErr    error
	}{
		{
			name: "explicit enabled id wins over default",
			registry: state.Registry{Default: "private", Sources: []state.Source{
				source("public", true),
				source("private", true),
			}},
			explicitID: "public",
			wantID:     "public",
		},
		{
			name: "explicit disabled id is unknown",
			registry: state.Registry{Sources: []state.Source{
				source("public", true),
				source("disabled", false),
			}},
			explicitID: "disabled",
			wantErr:    ErrUnknownSource,
		},
		{
			name: "explicit missing id is unknown",
			registry: state.Registry{Sources: []state.Source{
				source("public", true),
			}},
			explicitID: "missing",
			wantErr:    ErrUnknownSource,
		},
		{
			name: "default wins without explicit id",
			registry: state.Registry{Default: "private", Sources: []state.Source{
				source("public", true),
				source("private", true),
			}},
			wantID: "private",
		},
		{
			name: "sole enabled source wins without default",
			registry: state.Registry{Sources: []state.Source{
				source("public", true),
				source("disabled", false),
			}},
			wantID: "public",
		},
		{
			name: "zero enabled sources has no source",
			registry: state.Registry{Sources: []state.Source{
				source("disabled", false),
			}},
			wantErr: ErrNoSource,
		},
		{
			name: "multiple enabled sources without default has no source",
			registry: state.Registry{Sources: []state.Source{
				source("public", true),
				source("private", true),
			}},
			wantErr: ErrNoSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ActiveSource(tt.registry, tt.explicitID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ActiveSource() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ActiveSource() error = %v, want nil", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("ActiveSource() id = %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestActiveSourceErrorKindsStayDistinct(t *testing.T) {
	// M7 will map ErrNoSource to exit 3 and ErrUnknownSource to exit 4.
	if errors.Is(ErrNoSource, ErrUnknownSource) {
		t.Fatalf("ErrNoSource and ErrUnknownSource must remain distinct")
	}
}

func source(id string, enabled bool) state.Source {
	return state.Source{
		ID:      id,
		Enabled: enabled,
	}
}
