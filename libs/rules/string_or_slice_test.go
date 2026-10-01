package rules

import (
	"encoding/json"
	"testing"
)

func TestStringOrSlice_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantSlice []string
		wantErr   bool
	}{
		{
			name:      "single string",
			jsonInput: `"OUTLOOK.EXE"`,
			wantSlice: []string{"OUTLOOK.EXE"},
		},
		{
			name:      "array of strings",
			jsonInput: `["OUTLOOK.EXE", "olk.exe"]`,
			wantSlice: []string{"OUTLOOK.EXE", "olk.exe"},
		},
		{
			name:      "null value",
			jsonInput: `null`,
			wantSlice: nil,
		},
		{
			name:      "empty array",
			jsonInput: `[]`,
			wantSlice: []string{},
		},
		{
			name:      "invalid type number",
			jsonInput: `12345`,
			wantErr:   true,
		},
		{
			name:      "invalid type boolean",
			jsonInput: `true`,
			wantErr:   true,
		},
		{
			name:      "invalid type object",
			jsonInput: `{"key": "value"}`,
			wantErr:   true,
		},
		{
			name:      "invalid json syntax in array",
			jsonInput: `["unclosed`,
			wantErr:   true,
		},
		{
			name:      "invalid string escape sequence",
			jsonInput: `"invalid \u12"`,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var sos StringOrSlice
			err := json.Unmarshal([]byte(tc.jsonInput), &sos)
			if (err != nil) != tc.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if len(sos) != len(tc.wantSlice) {
					t.Fatalf("got slice length %d, want %d (%v)", len(sos), len(tc.wantSlice), sos)
				}
				for i := range sos {
					if sos[i] != tc.wantSlice[i] {
						t.Errorf("got slice[%d] = %q, want %q", i, sos[i], tc.wantSlice[i])
					}
				}
			}
		})
	}

	// Test direct UnmarshalJSON on empty byte slice
	var emptySOS StringOrSlice
	if err := emptySOS.UnmarshalJSON([]byte("   ")); err != nil || emptySOS != nil {
		t.Errorf("expected nil slice for empty bytes, got %v, %v", emptySOS, err)
	}
}
